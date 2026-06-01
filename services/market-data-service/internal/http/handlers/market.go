package handlers

import (
	"net/http"

	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/service"
)

type MarketHandler struct {
	service *service.MarketDataService
}

func NewMarketHandler(svc *service.MarketDataService) *MarketHandler {
	return &MarketHandler{service: svc}
}

func (h *MarketHandler) LatestTicks(w http.ResponseWriter, r *http.Request) {
	ticks, err := h.service.LatestTicks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticks": ticks})
}

func (h *MarketHandler) Symbols(w http.ResponseWriter, r *http.Request) {
	symbols, err := h.service.Symbols(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbols": symbols})
}
