package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/domain"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/service"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/tenant"
)

type NotificationHandler struct {
	service *service.NotificationService
}

type preferencesRequest struct {
	InAppEnabled   bool    `json:"inAppEnabled"`
	WebhookEnabled bool    `json:"webhookEnabled"`
	EmailEnabled   bool    `json:"emailEnabled"`
	WebhookURL     *string `json:"webhookUrl"`
	EmailAddress   *string `json:"emailAddress"`
	MinPriority    string  `json:"minPriority"`
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	filters := repository.ListFilters{
		UserID:   r.URL.Query().Get("userId"),
		Status:   r.URL.Query().Get("status"),
		Channel:  r.URL.Query().Get("channel"),
		Priority: r.URL.Query().Get("priority"),
		Limit:    boundedInt(r.URL.Query().Get("limit"), 50, 1, 200),
		Offset:   boundedInt(r.URL.Query().Get("offset"), 0, 0, 100000),
	}
	notifications, err := h.service.List(r.Context(), user, filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{
		"notifications": notifications,
		"limit":         filters.Limit,
		"offset":        filters.Offset,
	})
}

func (h *NotificationHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	notification, err := h.service.Get(r.Context(), user, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, notification)
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	notification, err := h.service.MarkRead(r.Context(), user, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, notification)
}

func (h *NotificationHandler) Retry(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	notification, err := h.service.Retry(r.Context(), user, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, notification)
}

func (h *NotificationHandler) Summary(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	summary, err := h.service.Summary(r.Context(), user)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, summary)
}

func (h *NotificationHandler) Preferences(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	prefs, err := h.service.Preferences(r.Context(), user)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, prefs)
}

func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req preferencesRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	prefs, err := h.service.UpdatePreferences(r.Context(), user, domain.Preferences{
		InAppEnabled:   req.InAppEnabled,
		WebhookEnabled: req.WebhookEnabled,
		EmailEnabled:   req.EmailEnabled,
		WebhookURL:     req.WebhookURL,
		EmailAddress:   req.EmailAddress,
		MinPriority:    req.MinPriority,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, prefs)
}

func userContext(r *http.Request) (service.UserContext, bool) {
	claims, ok := httpmiddleware.Claims(r.Context())
	if !ok {
		return service.UserContext{}, false
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = tenant.FromHeader(r)
	}
	return service.UserContext{UserID: claims.UserID, TenantID: tenant.Normalize(tenantID), Roles: claims.Roles}, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request")
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, repository.ErrNotFound):
		httpmiddleware.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, repository.ErrInvalidTransition):
		httpmiddleware.WriteError(w, http.StatusConflict, "invalid notification status transition")
	case errors.Is(err, service.ErrInvalidPreference):
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid notification preference")
	default:
		httpmiddleware.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}

func boundedInt(value string, fallback, min, max int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}
