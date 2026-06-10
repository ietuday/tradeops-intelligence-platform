package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
)

type RuleHandler struct {
	service *service.RuleConfigService
}

func NewRuleHandler(svc *service.RuleConfigService) *RuleHandler {
	return &RuleHandler{service: svc}
}

func (h *RuleHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rules, err := h.service.ListRules(r.Context(), user)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (h *RuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rule, err := h.service.GetRule(r.Context(), user, chi.URLParam(r, "ruleName"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, rule)
}

func (h *RuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request domain.UpdateRuleConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule, err := h.service.UpdateRule(r.Context(), user, chi.URLParam(r, "ruleName"), request, httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, rule)
}

func (h *RuleHandler) Enable(w http.ResponseWriter, r *http.Request) {
	h.setEnabled(w, r, h.service.EnableRule)
}

func (h *RuleHandler) Disable(w http.ResponseWriter, r *http.Request) {
	h.setEnabled(w, r, h.service.DisableRule)
}

func (h *RuleHandler) setEnabled(w http.ResponseWriter, r *http.Request, fn func(context.Context, service.UserContext, string, string) (domain.RuleConfig, error)) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rule, err := fn(r.Context(), user, chi.URLParam(r, "ruleName"), httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, rule)
}
