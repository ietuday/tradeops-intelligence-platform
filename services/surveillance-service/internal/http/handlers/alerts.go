package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
)

type AlertHandler struct {
	service *service.SurveillanceService
}

func NewAlertHandler(svc *service.SurveillanceService) *AlertHandler {
	return &AlertHandler{service: svc}
}

func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	filters := repository.AlertFilters{
		Status:    r.URL.Query().Get("status"),
		Severity:  r.URL.Query().Get("severity"),
		AlertType: r.URL.Query().Get("alertType"),
		UserID:    r.URL.Query().Get("userId"),
		Symbol:    r.URL.Query().Get("symbol"),
		Limit:     boundedInt(r.URL.Query().Get("limit"), 50, 1, 200),
		Offset:    boundedInt(r.URL.Query().Get("offset"), 0, 0, 100000),
	}
	alerts, err := h.service.ListAlerts(r.Context(), user, filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts,
		"limit":  filters.Limit,
		"offset": filters.Offset,
	})
}

func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	alert, err := h.service.GetAlert(r.Context(), user, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, alert)
}

func (h *AlertHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Acknowledge)
}

func (h *AlertHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Resolve)
}

func (h *AlertHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Dismiss)
}

func (h *AlertHandler) Summary(w http.ResponseWriter, r *http.Request) {
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

func (h *AlertHandler) transition(w http.ResponseWriter, r *http.Request, fn func(context.Context, service.UserContext, string, string) (domain.Alert, error)) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	alert, err := fn(r.Context(), user, chi.URLParam(r, "id"), httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, alert)
}

func userContext(r *http.Request) (service.UserContext, bool) {
	claims, ok := httpmiddleware.Claims(r.Context())
	if !ok {
		return service.UserContext{}, false
	}
	return service.UserContext{UserID: claims.UserID, Roles: claims.Roles}, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, repository.ErrNotFound):
		httpmiddleware.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, repository.ErrInvalidTransition):
		httpmiddleware.WriteError(w, http.StatusConflict, "invalid alert status transition")
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
