package service

import (
	"context"
	"errors"
	"strings"

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
	List(context.Context, repository.ListFilters) ([]domain.Notification, error)
	Get(context.Context, string) (domain.Notification, error)
	MarkRead(context.Context, string) (domain.Notification, error)
	Retry(context.Context, string) (domain.Notification, error)
	Summary(context.Context, string) (repository.Summary, error)
	Preferences(context.Context, string) (domain.Preferences, error)
	UpdatePreferences(context.Context, domain.Preferences) (domain.Preferences, error)
}

type NotificationService struct {
	store   Store
	metrics *observability.Metrics
}

func NewNotificationService(store Store, metrics *observability.Metrics) *NotificationService {
	return &NotificationService{store: store, metrics: metrics}
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
	return notification, nil
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
