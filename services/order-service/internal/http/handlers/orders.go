package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/service"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{service: svc}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	order, replay, err := h.service.CreateOrder(r.Context(), user, idempotencyKey, body, httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	status := http.StatusCreated
	if replay {
		status = http.StatusOK
	}
	httpmiddleware.WriteJSON(w, status, order)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orders, err := h.service.ListOrders(r.Context(), user)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	order, err := h.service.GetOrder(r.Context(), user, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	user, ok := userContext(r)
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	order, err := h.service.CancelOrder(r.Context(), user, chi.URLParam(r, "id"), httpmiddleware.GetCorrelationID(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, order)
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
	case errors.Is(err, service.ErrIdempotencyConflict):
		httpmiddleware.WriteError(w, http.StatusConflict, "idempotency key reused with different request payload")
	case errors.Is(err, service.ErrNotFound):
		httpmiddleware.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrOrderNotCancellable):
		httpmiddleware.WriteError(w, http.StatusConflict, "order cannot be cancelled")
	default:
		httpmiddleware.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
