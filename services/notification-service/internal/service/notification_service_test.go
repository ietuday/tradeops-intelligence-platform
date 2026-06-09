package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/repository"
)

func TestListScopesTraderToOwnNotifications(t *testing.T) {
	store := &fakeStore{}
	svc := NewNotificationService(store, observability.NewMetrics())

	_, err := svc.List(context.Background(), traderUser(), repository.ListFilters{UserID: "other-user", Limit: 50})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if store.lastFilters.UserID != traderUser().UserID {
		t.Fatalf("expected user scoped filter, got %q", store.lastFilters.UserID)
	}
}

func TestGetRejectsNotificationOwnedByAnotherUser(t *testing.T) {
	owner := "other-user"
	store := &fakeStore{notification: domain.Notification{ID: "n1", UserID: &owner}}
	svc := NewNotificationService(store, observability.NewMetrics())

	if _, err := svc.Get(context.Background(), traderUser(), "n1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestMarkReadAllowsOwner(t *testing.T) {
	owner := traderUser().UserID
	store := &fakeStore{notification: domain.Notification{ID: "n1", UserID: &owner, Status: domain.StatusSent}}
	svc := NewNotificationService(store, observability.NewMetrics())

	notification, err := svc.MarkRead(context.Background(), traderUser(), "n1")
	if err != nil {
		t.Fatalf("mark read failed: %v", err)
	}
	if notification.Status != domain.StatusRead || !store.markedRead {
		t.Fatalf("expected read notification, got %+v", notification)
	}
}

func TestRetryRequiresManagerRole(t *testing.T) {
	owner := traderUser().UserID
	store := &fakeStore{notification: domain.Notification{ID: "n1", UserID: &owner, Status: domain.StatusFailed}}
	svc := NewNotificationService(store, observability.NewMetrics())

	if _, err := svc.Retry(context.Background(), traderUser(), "n1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}

	notification, err := svc.Retry(context.Background(), managerUser(), "n1")
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if notification.Status != domain.StatusRetrying || !store.retried {
		t.Fatalf("expected retrying notification, got %+v", notification)
	}
}

func TestUpdatePreferencesNormalizesPriority(t *testing.T) {
	store := &fakeStore{}
	svc := NewNotificationService(store, observability.NewMetrics())

	prefs, err := svc.UpdatePreferences(context.Background(), traderUser(), domain.Preferences{MinPriority: "high", InAppEnabled: true})
	if err != nil {
		t.Fatalf("update preferences failed: %v", err)
	}
	if prefs.UserID != traderUser().UserID || prefs.MinPriority != domain.PriorityHigh {
		t.Fatalf("unexpected preferences: %+v", prefs)
	}
}

func TestUpdatePreferencesRejectsInvalidPriority(t *testing.T) {
	svc := NewNotificationService(&fakeStore{}, observability.NewMetrics())

	if _, err := svc.UpdatePreferences(context.Background(), traderUser(), domain.Preferences{MinPriority: "urgent"}); !errors.Is(err, ErrInvalidPreference) {
		t.Fatalf("expected invalid preference, got %v", err)
	}
}

func TestParseSurveillanceAlertEvent(t *testing.T) {
	payload := alertPayload(t, "surveillance.alert.created", "HIGH")
	event, err := ParseSurveillanceAlertEvent(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if event.AlertID != "alert-1" || event.EventType != "surveillance.alert.created" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestProcessEventCreatesNotificationFromSurveillanceAlert(t *testing.T) {
	store := &fakeStore{preferences: defaultPrefs("user-1")}
	publisher := &fakePublisher{}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), publisher, &fakeHTTPClient{}, nil, 2)

	err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "CRITICAL")})
	if err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected one created notification, got %d", len(store.created))
	}
	if store.created[0].Priority != domain.PriorityCritical || store.created[0].Channel != domain.ChannelInApp || store.created[0].Status != domain.StatusSent {
		t.Fatalf("unexpected notification: %+v", store.created[0])
	}
	if !publisher.hasEvent("notification.created") || !publisher.hasEvent("notification.sent") {
		t.Fatalf("expected created and sent events, got %+v", publisher.events)
	}
}

func TestSeverityToPriority(t *testing.T) {
	cases := map[string]string{
		"LOW":      domain.PriorityLow,
		"MEDIUM":   domain.PriorityMedium,
		"HIGH":     domain.PriorityHigh,
		"CRITICAL": domain.PriorityCritical,
		"unknown":  domain.PriorityMedium,
	}
	for severity, expected := range cases {
		if got := SeverityToPriority(severity); got != expected {
			t.Fatalf("expected %s for %s, got %s", expected, severity, got)
		}
	}
}

func TestProcessEventBadPayloadDoesNotCrash(t *testing.T) {
	store := &fakeStore{preferences: defaultPrefs("user-1")}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), nil, nil, nil, 2)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: []byte("{bad")}); err == nil {
		t.Fatal("expected bad payload to return an error for retry/DLQ handling")
	}
	if len(store.created) != 0 {
		t.Fatalf("bad payload should not create notifications")
	}
}

func TestProcessEventSkipsDuplicateSourceEventChannel(t *testing.T) {
	store := &fakeStore{preferences: defaultPrefs("user-1"), duplicate: true}
	publisher := &fakePublisher{}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), publisher, nil, nil, 2)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "HIGH")}); err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if len(store.created) != 0 {
		t.Fatalf("duplicate event should not create notifications, got %d", len(store.created))
	}
	if len(publisher.events) != 0 {
		t.Fatalf("duplicate event should not publish notification events, got %+v", publisher.events)
	}
}

func TestWebhookSuccess(t *testing.T) {
	prefs := defaultPrefs("user-1")
	url := "http://webhook.test/notify"
	prefs.WebhookEnabled = true
	prefs.WebhookURL = &url
	store := &fakeStore{preferences: prefs}
	client := &fakeHTTPClient{statuses: []int{204}}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), nil, client, nil, 2)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "HIGH")}); err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if len(store.created) != 2 {
		t.Fatalf("expected in-app and webhook notifications, got %d", len(store.created))
	}
	if got := store.statusByID[store.created[1].ID]; got != domain.StatusSent {
		t.Fatalf("expected webhook sent, got %s", got)
	}
	if len(store.attempts) != 1 || store.attempts[0].Status != domain.StatusSent {
		t.Fatalf("expected sent attempt, got %+v", store.attempts)
	}
}

func TestWebhookFailure(t *testing.T) {
	prefs := defaultPrefs("user-1")
	url := "http://webhook.test/notify"
	prefs.InAppEnabled = false
	prefs.WebhookEnabled = true
	prefs.WebhookURL = &url
	store := &fakeStore{preferences: prefs}
	client := &fakeHTTPClient{statuses: []int{500, 503}}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), nil, client, nil, 2)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "HIGH")}); err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if got := store.statusByID[store.created[0].ID]; got != domain.StatusFailed {
		t.Fatalf("expected webhook failed, got %s", got)
	}
	if len(store.attempts) != 2 {
		t.Fatalf("expected two attempts, got %d", len(store.attempts))
	}
}

func TestMockEmailSuccess(t *testing.T) {
	prefs := defaultPrefs("user-1")
	email := "demo@example.com"
	prefs.InAppEnabled = false
	prefs.EmailEnabled = true
	prefs.EmailAddress = &email
	store := &fakeStore{preferences: prefs}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), nil, nil, nil, 2)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "LOW")}); err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if len(store.created) != 1 || store.created[0].Channel != domain.ChannelEmail || store.created[0].Status != domain.StatusSent {
		t.Fatalf("expected sent email notification, got %+v", store.created)
	}
	if len(store.attempts) != 1 || store.attempts[0].Status != domain.StatusSent {
		t.Fatalf("expected email attempt, got %+v", store.attempts)
	}
}

func TestWebhookRetryBehavior(t *testing.T) {
	prefs := defaultPrefs("user-1")
	url := "http://webhook.test/notify"
	prefs.InAppEnabled = false
	prefs.WebhookEnabled = true
	prefs.WebhookURL = &url
	store := &fakeStore{preferences: prefs}
	client := &fakeHTTPClient{statuses: []int{500, 200}}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), nil, client, nil, 3)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "HIGH")}); err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if len(store.attempts) != 2 {
		t.Fatalf("expected retry then success, got %+v", store.attempts)
	}
	if store.attempts[0].Status != domain.StatusFailed || store.attempts[1].Status != domain.StatusSent {
		t.Fatalf("unexpected attempts: %+v", store.attempts)
	}
}

func TestWebhookTimeoutFailure(t *testing.T) {
	prefs := defaultPrefs("user-1")
	url := "http://webhook.test/notify"
	prefs.InAppEnabled = false
	prefs.WebhookEnabled = true
	prefs.WebhookURL = &url
	store := &fakeStore{preferences: prefs}
	client := &fakeHTTPClient{errs: []error{context.DeadlineExceeded, context.DeadlineExceeded}}
	svc := NewNotificationServiceWithPublisher(store, observability.NewMetrics(), nil, client, nil, 2)

	if err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "surveillance.alert.created", Value: alertPayload(t, "surveillance.alert.created", "HIGH")}); err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if got := store.statusByID[store.created[0].ID]; got != domain.StatusFailed {
		t.Fatalf("expected webhook timeout to fail after retries, got %s", got)
	}
	if len(store.attempts) != 2 {
		t.Fatalf("expected two failed timeout attempts, got %d", len(store.attempts))
	}
}

type fakeStore struct {
	notification domain.Notification
	preferences  domain.Preferences
	lastFilters  repository.ListFilters
	markedRead   bool
	retried      bool
	duplicate    bool
	created      []domain.Notification
	attempts     []domain.DeliveryAttempt
	statusByID   map[string]string
}

func (s *fakeStore) Create(_ context.Context, notification domain.Notification) (domain.Notification, error) {
	s.created = append(s.created, notification)
	if s.statusByID == nil {
		s.statusByID = map[string]string{}
	}
	s.statusByID[notification.ID] = notification.Status
	return notification, nil
}

func (s *fakeStore) DuplicateExists(context.Context, string, string, string, string) (bool, error) {
	return s.duplicate, nil
}

func (s *fakeStore) List(_ context.Context, filters repository.ListFilters) ([]domain.Notification, error) {
	s.lastFilters = filters
	return []domain.Notification{s.notification}, nil
}

func (s *fakeStore) Get(_ context.Context, _, _ string) (domain.Notification, error) {
	if s.notification.ID == "" {
		return domain.Notification{}, repository.ErrNotFound
	}
	return s.notification, nil
}

func (s *fakeStore) MarkRead(_ context.Context, _, _ string) (domain.Notification, error) {
	s.markedRead = true
	s.notification.Status = domain.StatusRead
	now := time.Now().UTC()
	s.notification.ReadAt = &now
	return s.notification, nil
}

func (s *fakeStore) Retry(_ context.Context, _, _ string) (domain.Notification, error) {
	s.retried = true
	s.notification.Status = domain.StatusRetrying
	return s.notification, nil
}

func (s *fakeStore) UpdateStatus(_ context.Context, _ string, id string, status string) (domain.Notification, error) {
	if s.statusByID == nil {
		s.statusByID = map[string]string{}
	}
	s.statusByID[id] = status
	for i := range s.created {
		if s.created[i].ID == id {
			s.created[i].Status = status
			return s.created[i], nil
		}
	}
	s.notification.Status = status
	return s.notification, nil
}

func (s *fakeStore) RecordAttempt(_ context.Context, attempt domain.DeliveryAttempt) (domain.DeliveryAttempt, error) {
	s.attempts = append(s.attempts, attempt)
	return attempt, nil
}

func (s *fakeStore) NextAttemptNumber(context.Context, string) (int, error) {
	return len(s.attempts) + 1, nil
}

func (s *fakeStore) Summary(context.Context, string, string) (repository.Summary, error) {
	return repository.Summary{Unread: 1}, nil
}

func (s *fakeStore) Preferences(context.Context, string, string) (domain.Preferences, error) {
	if s.preferences.UserID == "" {
		s.preferences = defaultPrefs("user-1")
	}
	return s.preferences, nil
}

func (s *fakeStore) UpdatePreferences(_ context.Context, prefs domain.Preferences) (domain.Preferences, error) {
	s.preferences = prefs
	return prefs, nil
}

func traderUser() UserContext {
	return UserContext{UserID: "user-1", TenantID: "default-tenant", Roles: []string{"trader"}}
}

func managerUser() UserContext {
	return UserContext{UserID: "manager-1", TenantID: "default-tenant", Roles: []string{"risk_manager"}}
}

func defaultPrefs(userID string) domain.Preferences {
	return domain.Preferences{TenantID: "default-tenant", UserID: userID, InAppEnabled: true, MinPriority: domain.PriorityLow}
}

func alertPayload(t *testing.T, eventType, severity string) []byte {
	t.Helper()
	userID := "user-1"
	symbol := "AAPL"
	payload, err := json.Marshal(domain.SurveillanceAlertEvent{
		EventID: "event-1", EventType: eventType, AlertID: "alert-1", AlertType: "LargeOrderRule",
		TenantID: "default-tenant",
		Severity: severity, EntityType: "ORDER", EntityID: "order-1", UserID: &userID, Symbol: &symbol,
		Status: "OPEN", Metadata: map[string]any{"notional": 150000}, OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

type fakePublisher struct {
	events []domain.NotificationEvent
}

func (p *fakePublisher) Publish(_ context.Context, event domain.NotificationEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakePublisher) hasEvent(eventType string) bool {
	for _, event := range p.events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

type fakeHTTPClient struct {
	statuses []int
	errs     []error
	calls    int
}

func (c *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.calls++
	if len(c.errs) >= c.calls && c.errs[c.calls-1] != nil {
		return nil, c.errs[c.calls-1]
	}
	status := http.StatusOK
	if len(c.statuses) >= c.calls {
		status = c.statuses[c.calls-1]
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
}
