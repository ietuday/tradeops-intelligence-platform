package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/service"
)

type AuditHandler struct {
	service *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{service: svc}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	filters := filtersFromRequest(r)
	logs, err := h.service.ListAuditLogs(r.Context(), user, filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{
		"auditLogs": logs,
		"limit":     filters.Limit,
		"offset":    filters.Offset,
	})
}

func (h *AuditHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	log, err := h.service.GetAuditLog(r.Context(), user, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, log)
}

func (h *AuditHandler) Summary(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	summary, err := h.service.Summary(r.Context(), user, filtersFromRequest(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, summary)
}

func (h *AuditHandler) Export(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	result, err := h.service.Export(r.Context(), user, filtersFromRequest(r), r.URL.Query().Get("format"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+result.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Body)
}

func userContext(r *http.Request) (service.UserContext, bool) {
	claims, ok := httpmiddleware.Claims(r.Context())
	if !ok {
		return service.UserContext{}, false
	}
	return service.UserContext{UserID: claims.UserID, Roles: claims.Roles}, true
}

func filtersFromRequest(r *http.Request) repository.ListFilters {
	query := r.URL.Query()
	return repository.ListFilters{
		EventType:     query.Get("eventType"),
		ServiceName:   query.Get("serviceName"),
		ActorUserID:   query.Get("actorUserId"),
		EntityType:    query.Get("entityType"),
		EntityID:      query.Get("entityId"),
		Action:        query.Get("action"),
		Severity:      query.Get("severity"),
		CorrelationID: query.Get("correlationId"),
		From:          parseTime(query.Get("from")),
		To:            parseTime(query.Get("to")),
		Limit:         boundedInt(query.Get("limit"), 50, 1, 1000),
		Offset:        boundedInt(query.Get("offset"), 0, 0, 100000),
	}
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, repository.ErrNotFound):
		httpmiddleware.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrInvalidExportFormat):
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid export format")
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
