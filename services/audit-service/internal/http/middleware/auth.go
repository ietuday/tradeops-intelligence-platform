package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/security"
)

type claimsKey struct{}

func Auth(validator *security.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if claims, ok := trustedHeaderClaims(r); ok {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
				return
			}
			if !strings.HasPrefix(header, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			claims, err := validator.Validate(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}
}

func ServiceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if validServiceAuth(r) {
			next.ServeHTTP(w, r)
			return
		}
		WriteError(w, http.StatusUnauthorized, "unauthorized")
	})
}

func Claims(ctx context.Context) (*security.Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*security.Claims)
	return claims, ok
}

func trustedHeaderClaims(r *http.Request) (*security.Claims, bool) {
	if !validServiceAuth(r) {
		return nil, false
	}
	userID := strings.TrimSpace(r.Header.Get("X-TradeOps-User-Id"))
	subject := strings.TrimSpace(r.Header.Get("X-TradeOps-Subject"))
	if userID == "" {
		userID = subject
	}
	tenantID := strings.TrimSpace(r.Header.Get("X-TradeOps-Tenant-Id"))
	if userID == "" || tenantID == "" {
		return nil, false
	}
	return &security.Claims{
		UserID:   userID,
		TenantID: tenantID,
		Roles:    splitHeaderList(r.Header.Get("X-TradeOps-Roles")),
	}, true
}

func validServiceAuth(r *http.Request) bool {
	if os.Getenv("SERVICE_AUTH_ENABLED") == "false" {
		return true
	}
	expected := os.Getenv("SERVICE_AUTH_SHARED_SECRET")
	if expected == "" {
		expected = "local-dev-service-secret"
	}
	token := r.Header.Get("X-TradeOps-Service-Token")
	caller := r.Header.Get("X-TradeOps-Service-Name")
	if caller == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func splitHeaderList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
