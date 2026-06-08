package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/repository"
)

var ErrForbidden = errors.New("forbidden")
var ErrInvalidPreference = errors.New("invalid notification preference")

type UserContext struct {
	UserID string
	Roles  []string
}

type Store interface {
	Create(context.Context, domain.Notification) (domain.Notification, error)
	DuplicateExists(context.Context, string, string, string) (bool, error)
	List(context.Context, repository.ListFilters) ([]domain.Notification, error)
	Get(context.Context, string) (domain.Notification, error)
	MarkRead(context.Context, string) (domain.Notification, error)
	Retry(context.Context, string) (domain.Notification, error)
	UpdateStatus(context.Context, string, string) (domain.Notification, error)
	RecordAttempt(context.Context, domain.DeliveryAttempt) (domain.DeliveryAttempt, error)
	NextAttemptNumber(context.Context, string) (int, error)
	Summary(context.Context, string) (repository.Summary, error)
	Preferences(context.Context, string) (domain.Preferences, error)
	UpdatePreferences(context.Context, domain.Preferences) (domain.Preferences, error)
}

type Publisher interface {
	Publish(context.Context, domain.NotificationEvent) error
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, domain.NotificationEvent) error { return nil }

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type NotificationService struct {
	store             Store
	metrics           *observability.Metrics
	publisher         Publisher
	httpClient        HTTPClient
	logger            *slog.Logger
	webhookMaxRetries int
}

func NewNotificationService(store Store, metrics *observability.Metrics) *NotificationService {
	return NewNotificationServiceWithPublisher(store, metrics, noopPublisher{}, http.DefaultClient, slog.Default(), 3)
}

func NewNotificationServiceWithPublisher(store Store, metrics *observability.Metrics, publisher Publisher, httpClient HTTPClient, logger *slog.Logger, webhookMaxRetries int) *NotificationService {
	if publisher == nil {
		publisher = noopPublisher{}
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = slog.Default()
	}
	if webhookMaxRetries < 1 {
		webhookMaxRetries = 1
	}
	return &NotificationService{store: store, metrics: metrics, publisher: publisher, httpClient: httpClient, logger: logger, webhookMaxRetries: webhookMaxRetries}
}

func (s *NotificationService) ProcessEvent(ctx context.Context, event domain.SourceEvent) error {
	alert, err := ParseSurveillanceAlertEvent(event.Value)
	if err != nil {
		s.metrics.EventsFailed.Inc()
		return err
	}
	if alert.EventType == "" {
		alert.EventType = event.Topic
	}
	if alert.CorrelationID == "" {
		alert.CorrelationID = event.CorrelationID
	}
	if alert.UserID == nil || *alert.UserID == "" {
		s.metrics.EventsProcessed.Inc()
		return nil
	}
	if err := s.createFromAlert(ctx, alert); err != nil {
		s.metrics.EventsFailed.Inc()
		return err
	}
	s.metrics.EventsProcessed.Inc()
	return nil
}

func ParseSurveillanceAlertEvent(payload []byte) (domain.SurveillanceAlertEvent, error) {
	var event domain.SurveillanceAlertEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return event, err
	}
	if event.EventID == "" || event.AlertID == "" || event.EventType == "" {
		return event, errors.New("invalid surveillance alert event")
	}
	return event, nil
}

func (s *NotificationService) createFromAlert(ctx context.Context, alert domain.SurveillanceAlertEvent) error {
	prefs, err := s.store.Preferences(ctx, *alert.UserID)
	if err != nil {
		return err
	}
	base := notificationFromAlert(alert)
	if prefs.InAppEnabled {
		if err := s.sendInApp(ctx, base); err != nil {
			return err
		}
	}
	if prefs.EmailEnabled {
		notification := base
		notification.ID = uuid.NewString()
		notification.Channel = domain.ChannelEmail
		if prefs.EmailAddress != nil {
			notification.Metadata["emailAddress"] = *prefs.EmailAddress
		}
		if err := s.sendEmail(ctx, notification); err != nil {
			return err
		}
	}
	if prefs.WebhookEnabled && prefs.WebhookURL != nil && strings.TrimSpace(*prefs.WebhookURL) != "" {
		notification := base
		notification.ID = uuid.NewString()
		notification.Channel = domain.ChannelWebhook
		notification.Metadata["webhookUrl"] = *prefs.WebhookURL
		if err := s.sendWebhook(ctx, notification, *prefs.WebhookURL); err != nil {
			return err
		}
	}
	return nil
}

func notificationFromAlert(alert domain.SurveillanceAlertEvent) domain.Notification {
	now := time.Now().UTC()
	title := fmt.Sprintf("Surveillance alert: %s", alert.AlertType)
	message := fmt.Sprintf("%s alert %s is %s", alert.AlertType, alert.AlertID, alert.Status)
	switch alert.EventType {
	case "surveillance.alert.created":
		message = fmt.Sprintf("%s alert created for %s %s", alert.AlertType, alert.EntityType, alert.EntityID)
	case "surveillance.alert.acknowledged":
		message = fmt.Sprintf("%s alert acknowledged", alert.AlertType)
	case "surveillance.alert.resolved":
		message = fmt.Sprintf("%s alert resolved", alert.AlertType)
	case "surveillance.alert.dismissed":
		message = fmt.Sprintf("%s alert dismissed", alert.AlertType)
	}
	metadata := map[string]any{
		"sourceEventId": alert.EventID,
		"alertId":       alert.AlertID,
		"alertType":     alert.AlertType,
		"entityType":    alert.EntityType,
		"entityId":      alert.EntityID,
		"alertStatus":   alert.Status,
		"eventType":     alert.EventType,
	}
	if alert.Symbol != nil {
		metadata["symbol"] = *alert.Symbol
	}
	for key, value := range alert.Metadata {
		metadata[key] = value
	}
	return domain.Notification{
		ID:        uuid.NewString(),
		UserID:    alert.UserID,
		Channel:   domain.ChannelInApp,
		Priority:  SeverityToPriority(alert.Severity),
		Status:    domain.StatusPending,
		Title:     title,
		Message:   message,
		Source:    "surveillance-service",
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func SeverityToPriority(severity string) string {
	switch strings.ToUpper(severity) {
	case "LOW":
		return domain.PriorityLow
	case "MEDIUM":
		return domain.PriorityMedium
	case "HIGH":
		return domain.PriorityHigh
	case "CRITICAL":
		return domain.PriorityCritical
	default:
		return domain.PriorityMedium
	}
}

func (s *NotificationService) sendInApp(ctx context.Context, notification domain.Notification) error {
	notification.Channel = domain.ChannelInApp
	notification.Status = domain.StatusSent
	if duplicate, err := s.duplicateNotification(ctx, notification); err != nil {
		return err
	} else if duplicate {
		s.metrics.DuplicateSkipped.WithLabelValues(metadataString(notification.Metadata, "eventType")).Inc()
		return nil
	}
	created, err := s.store.Create(ctx, notification)
	if err != nil {
		return err
	}
	s.metrics.NotificationsCreated.Inc()
	s.metrics.StatusUpdates.Inc()
	s.publish(ctx, "notification.created", created, "")
	s.publish(ctx, "notification.sent", created, "")
	return nil
}

func (s *NotificationService) sendEmail(ctx context.Context, notification domain.Notification) error {
	start := time.Now()
	notification.Channel = domain.ChannelEmail
	notification.Status = domain.StatusSent
	if duplicate, err := s.duplicateNotification(ctx, notification); err != nil {
		return err
	} else if duplicate {
		s.metrics.DuplicateSkipped.WithLabelValues(metadataString(notification.Metadata, "eventType")).Inc()
		return nil
	}
	created, err := s.store.Create(ctx, notification)
	if err != nil {
		return err
	}
	s.logger.Info("mock email notification delivered", "notificationId", created.ID, "userId", ptrValue(created.UserID))
	if err := s.recordAttempt(ctx, created.ID, domain.ChannelEmail, domain.StatusSent, nil); err != nil {
		return err
	}
	s.metrics.NotificationsCreated.Inc()
	s.metrics.StatusUpdates.Inc()
	s.metrics.ObserveDelivery(start)
	s.publish(ctx, "notification.created", created, "")
	s.publish(ctx, "notification.sent", created, "")
	return nil
}

func (s *NotificationService) sendWebhook(ctx context.Context, notification domain.Notification, webhookURL string) error {
	start := time.Now()
	notification.Channel = domain.ChannelWebhook
	notification.Status = domain.StatusPending
	if duplicate, err := s.duplicateNotification(ctx, notification); err != nil {
		return err
	} else if duplicate {
		s.metrics.DuplicateSkipped.WithLabelValues(metadataString(notification.Metadata, "eventType")).Inc()
		return nil
	}
	created, err := s.store.Create(ctx, notification)
	if err != nil {
		return err
	}
	s.metrics.NotificationsCreated.Inc()
	s.publish(ctx, "notification.created", created, "")

	payload, err := json.Marshal(created)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 1; attempt <= s.webhookMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err == nil && resp.Body != nil {
			resp.Body.Close()
		}
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if recErr := s.recordAttempt(ctx, created.ID, domain.ChannelWebhook, domain.StatusSent, nil); recErr != nil {
				return recErr
			}
			sent, err := s.store.UpdateStatus(ctx, created.ID, domain.StatusSent)
			if err != nil {
				return err
			}
			s.metrics.StatusUpdates.Inc()
			s.metrics.ObserveDelivery(start)
			s.publish(ctx, "notification.sent", sent, "")
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}
		msg := lastErr.Error()
		if recErr := s.recordAttempt(ctx, created.ID, domain.ChannelWebhook, domain.StatusFailed, &msg); recErr != nil {
			return recErr
		}
		if attempt < s.webhookMaxRetries {
			time.Sleep(time.Duration(attempt) * 50 * time.Millisecond)
		}
	}
	failed, err := s.store.UpdateStatus(ctx, created.ID, domain.StatusFailed)
	if err != nil {
		return err
	}
	s.metrics.DeliveryFailures.Inc()
	s.metrics.StatusUpdates.Inc()
	s.metrics.ObserveDelivery(start)
	s.publish(ctx, "notification.failed", failed, "")
	return nil
}

func (s *NotificationService) List(ctx context.Context, user UserContext, filters repository.ListFilters) ([]domain.Notification, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	if !canViewAll(user.Roles) || filters.UserID == "" {
		filters.UserID = user.UserID
	}
	return s.store.List(ctx, filters)
}

func (s *NotificationService) Get(ctx context.Context, user UserContext, id string) (domain.Notification, error) {
	if !canView(user.Roles) {
		return domain.Notification{}, ErrForbidden
	}
	notification, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	if !canAccess(user, notification) {
		return domain.Notification{}, ErrForbidden
	}
	return notification, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, user UserContext, id string) (domain.Notification, error) {
	if _, err := s.Get(ctx, user, id); err != nil {
		return domain.Notification{}, err
	}
	notification, err := s.store.MarkRead(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	s.metrics.NotificationsMarkedRead.Inc()
	s.metrics.StatusUpdates.Inc()
	s.publish(ctx, "notification.read", notification, "")
	return notification, nil
}

func (s *NotificationService) Retry(ctx context.Context, user UserContext, id string) (domain.Notification, error) {
	if !canManage(user.Roles) {
		return domain.Notification{}, ErrForbidden
	}
	if _, err := s.store.Get(ctx, id); err != nil {
		return domain.Notification{}, err
	}
	notification, err := s.store.Retry(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	s.metrics.NotificationRetries.Inc()
	s.metrics.StatusUpdates.Inc()
	s.publish(ctx, "notification.retry_requested", notification, "")
	return notification, nil
}

func (s *NotificationService) recordAttempt(ctx context.Context, notificationID, channel, status string, errorMessage *string) error {
	next, err := s.store.NextAttemptNumber(ctx, notificationID)
	if err != nil {
		return err
	}
	_, err = s.store.RecordAttempt(ctx, domain.DeliveryAttempt{
		ID:             uuid.NewString(),
		NotificationID: notificationID,
		Channel:        channel,
		Status:         status,
		AttemptNumber:  next,
		ErrorMessage:   errorMessage,
		AttemptedAt:    time.Now().UTC(),
	})
	if err == nil {
		s.metrics.DeliveryAttempts.Inc()
	}
	return err
}

func (s *NotificationService) duplicateNotification(ctx context.Context, notification domain.Notification) (bool, error) {
	sourceEventID := metadataString(notification.Metadata, "sourceEventId")
	eventType := metadataString(notification.Metadata, "eventType")
	if sourceEventID == "" || eventType == "" {
		return false, nil
	}
	return s.store.DuplicateExists(ctx, sourceEventID, eventType, notification.Channel)
}

func (s *NotificationService) publish(ctx context.Context, eventType string, notification domain.Notification, correlationID string) {
	_ = s.publisher.Publish(ctx, domain.NotificationEvent{
		EventID:        uuid.NewString(),
		EventType:      eventType,
		NotificationID: notification.ID,
		UserID:         notification.UserID,
		Channel:        notification.Channel,
		Priority:       notification.Priority,
		Status:         notification.Status,
		Source:         notification.Source,
		Metadata:       notification.Metadata,
		OccurredAt:     time.Now().UTC(),
		CorrelationID:  correlationID,
	})
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return ""
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *NotificationService) Summary(ctx context.Context, user UserContext) (repository.Summary, error) {
	if !canView(user.Roles) {
		return repository.Summary{}, ErrForbidden
	}
	return s.store.Summary(ctx, user.UserID)
}

func (s *NotificationService) Preferences(ctx context.Context, user UserContext) (domain.Preferences, error) {
	if user.UserID == "" {
		return domain.Preferences{}, ErrForbidden
	}
	return s.store.Preferences(ctx, user.UserID)
}

func (s *NotificationService) UpdatePreferences(ctx context.Context, user UserContext, prefs domain.Preferences) (domain.Preferences, error) {
	if user.UserID == "" {
		return domain.Preferences{}, ErrForbidden
	}
	prefs.UserID = user.UserID
	prefs.MinPriority = strings.ToUpper(strings.TrimSpace(prefs.MinPriority))
	if prefs.MinPriority == "" {
		prefs.MinPriority = domain.PriorityLow
	}
	if !validPriority(prefs.MinPriority) {
		return domain.Preferences{}, ErrInvalidPreference
	}
	updated, err := s.store.UpdatePreferences(ctx, prefs)
	if err != nil {
		return domain.Preferences{}, err
	}
	s.metrics.PreferencesUpdated.Inc()
	return updated, nil
}

func canAccess(user UserContext, notification domain.Notification) bool {
	if canViewAll(user.Roles) {
		return true
	}
	return notification.UserID != nil && *notification.UserID == user.UserID
}

func canView(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager", "analyst", "viewer", "trader":
			return true
		}
	}
	return false
}

func canViewAll(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager":
			return true
		}
	}
	return false
}

func canManage(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager":
			return true
		}
	}
	return false
}

func validPriority(priority string) bool {
	switch priority {
	case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityCritical:
		return true
	default:
		return false
	}
}
