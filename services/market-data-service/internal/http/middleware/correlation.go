package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const CorrelationIDHeader = "X-Correlation-ID"

type correlationIDKey struct{}

func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		w.Header().Set(CorrelationIDHeader, correlationID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), correlationIDKey{}, correlationID)))
	})
}

func GetCorrelationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationIDKey{}).(string)
	return value
}
