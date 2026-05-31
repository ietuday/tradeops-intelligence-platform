package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/security"
)

type claimsKey struct{}

func Auth(tokenManager *security.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			claims, err := tokenManager.ValidateAccessToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}
}

func Claims(ctx context.Context) (*security.Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*security.Claims)
	return claims, ok
}
