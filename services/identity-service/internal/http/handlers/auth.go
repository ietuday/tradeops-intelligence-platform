package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type userResponse struct {
	ID       string   `json:"id"`
	TenantID string   `json:"tenantId"`
	Email    string   `json:"email"`
	FullName string   `json:"fullName"`
	Roles    []string `json:"roles"`
}

type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int    `json:"expiresIn"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" || len(req.Password) < 8 || strings.TrimSpace(req.FullName) == "" {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	user, err := h.auth.Register(r.Context(), req.Email, req.Password, req.FullName, auditContext(r))
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			httpmiddleware.WriteError(w, http.StatusConflict, "email already registered")
			return
		}
		httpmiddleware.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusCreated, userResponse{ID: user.ID, TenantID: user.TenantID, Email: user.Email, FullName: user.FullName, Roles: user.Roles})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	pair, err := h.auth.Login(r.Context(), req.Email, req.Password, auditContext(r))
	if err != nil {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, tokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, TokenType: pair.TokenType, ExpiresIn: pair.ExpiresIn})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	pair, err := h.auth.Refresh(r.Context(), req.RefreshToken, auditContext(r))
	if err != nil {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, tokenResponse{AccessToken: pair.AccessToken, TokenType: pair.TokenType, ExpiresIn: pair.ExpiresIn})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		httpmiddleware.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}
	_ = h.auth.Logout(r.Context(), req.RefreshToken, auditContext(r))
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.Claims(r.Context())
	if !ok {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.auth.Me(r.Context(), claims.UserID)
	if err != nil {
		httpmiddleware.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, userResponse{ID: user.ID, TenantID: user.TenantID, Email: user.Email, FullName: user.FullName, Roles: user.Roles})
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

func auditContext(r *http.Request) service.AuditContext {
	return service.AuditContext{
		CorrelationID: httpmiddleware.GetCorrelationID(r.Context()),
		IPAddress:     clientIP(r),
		UserAgent:     r.UserAgent(),
	}
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
