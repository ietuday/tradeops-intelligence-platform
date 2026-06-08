package service

import (
	"context"
	"errors"
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

type fakeStore struct {
	notification domain.Notification
	preferences  domain.Preferences
	lastFilters  repository.ListFilters
	markedRead   bool
	retried      bool
}

func (s *fakeStore) List(_ context.Context, filters repository.ListFilters) ([]domain.Notification, error) {
	s.lastFilters = filters
	return []domain.Notification{s.notification}, nil
}

func (s *fakeStore) Get(_ context.Context, _ string) (domain.Notification, error) {
	if s.notification.ID == "" {
		return domain.Notification{}, repository.ErrNotFound
	}
	return s.notification, nil
}

func (s *fakeStore) MarkRead(_ context.Context, _ string) (domain.Notification, error) {
	s.markedRead = true
	s.notification.Status = domain.StatusRead
	now := time.Now().UTC()
	s.notification.ReadAt = &now
	return s.notification, nil
}

func (s *fakeStore) Retry(_ context.Context, _ string) (domain.Notification, error) {
	s.retried = true
	s.notification.Status = domain.StatusRetrying
	return s.notification, nil
}

func (s *fakeStore) Summary(context.Context, string) (repository.Summary, error) {
	return repository.Summary{Unread: 1}, nil
}

func (s *fakeStore) Preferences(context.Context, string) (domain.Preferences, error) {
	return s.preferences, nil
}

func (s *fakeStore) UpdatePreferences(_ context.Context, prefs domain.Preferences) (domain.Preferences, error) {
	s.preferences = prefs
	return prefs, nil
}

func traderUser() UserContext {
	return UserContext{UserID: "user-1", Roles: []string{"trader"}}
}

func managerUser() UserContext {
	return UserContext{UserID: "manager-1", Roles: []string{"risk_manager"}}
}
