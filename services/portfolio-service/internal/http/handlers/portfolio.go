package handlers

import (
	"errors"
	"net/http"

	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/service"
)

type PortfolioHandler struct {
	service *service.PortfolioService
}

func NewPortfolioHandler(svc *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{service: svc}
}

func (h *PortfolioHandler) Portfolio(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	portfolio, err := h.service.Portfolio(r.Context(), user)
	writeResult(w, portfolio, err)
}

func (h *PortfolioHandler) Holdings(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	holdings, err := h.service.Holdings(r.Context(), user)
	writeResult(w, map[string]any{"holdings": holdings}, err)
}

func (h *PortfolioHandler) Snapshots(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	snapshots, err := h.service.Snapshots(r.Context(), user)
	writeResult(w, map[string]any{"snapshots": snapshots}, err)
}

func (h *PortfolioHandler) Exposure(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	exposure, err := h.service.Exposure(r.Context(), user)
	writeResult(w, exposure, err)
}

func (h *PortfolioHandler) PnL(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	pnl, err := h.service.PnL(r.Context(), user)
	writeResult(w, pnl, err)
}

func userContext(r *http.Request) (service.UserContext, bool) {
	claims, ok := httpmiddleware.Claims(r.Context())
	if !ok {
		return service.UserContext{}, false
	}
	return service.UserContext{UserID: claims.UserID, Roles: claims.Roles}, true
}

func writeResult(w http.ResponseWriter, value any, err error) {
	if err == nil {
		httpmiddleware.WriteJSON(w, http.StatusOK, value)
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		httpmiddleware.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	httpmiddleware.WriteError(w, http.StatusInternalServerError, "internal error")
}
