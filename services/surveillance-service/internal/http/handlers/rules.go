package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/simulation"
)

type RuleHandler struct {
	service    *service.RuleConfigService
	simulation *simulation.Service
}

func NewRuleHandler(svc *service.RuleConfigService, simulationSvc *simulation.Service) *RuleHandler {
	return &RuleHandler{service: svc, simulation: simulationSvc}
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

func (h *RuleHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !canSimulate(user.Roles) {
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	var request simulation.RuleSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	pathRule := chi.URLParam(r, "ruleName")
	if pathRule != "" {
		request.RuleName = pathRule
	}
	if request.TenantID != "" && request.TenantID != user.TenantID && !hasRole(user.Roles, "trading_admin") {
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	result, err := h.simulation.Simulate(r.Context(), request, httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeSimulationError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, result)
}

func (h *RuleHandler) SimulateBulk(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !canSimulate(user.Roles) {
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	var request simulation.RuleSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if request.TenantID != "" && request.TenantID != user.TenantID && !hasRole(user.Roles, "trading_admin") {
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	result, err := h.simulation.SimulateBulk(r.Context(), request, httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeSimulationError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, result)
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

func writeSimulationError(w http.ResponseWriter, err error) {
	if errors.Is(err, simulation.ErrInvalidRequest) {
		httpmiddleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeServiceError(w, err)
}

func canSimulate(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager", "analyst", "viewer":
			return true
		}
	}
	return false
}

func hasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}
