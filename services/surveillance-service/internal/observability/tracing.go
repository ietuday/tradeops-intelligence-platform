package observability

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func SetupTracing(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	if strings.ToLower(os.Getenv("OTEL_ENABLED")) != "true" {
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return func(context.Context) error { return nil }, nil
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return func(context.Context) error { return nil }, fmt.Errorf("otel exporter: %w", err)
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(os.Getenv("OTEL_SERVICE_VERSION")),
			attribute.String("tradeops.service", serviceName),
		)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return provider.Shutdown, nil
}

func HTTPHandler(serviceName string, handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, serviceName+".http", otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		if route := chi.RouteContext(r.Context()).RoutePattern(); route != "" {
			return r.Method + " " + route
		}
		return r.Method + " " + r.URL.Path
	}))
}

func TraceAttributes(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				span := trace.SpanFromContext(r.Context())
				span.SetAttributes(
					attribute.String("tradeops.service", serviceName),
					attribute.String("service.name", serviceName),
					attribute.String("correlation.id", r.Header.Get("X-Correlation-ID")),
					attribute.String("tenant.id", r.Header.Get("X-Tenant-ID")),
				)
				if userID := jwtSubject(r.Header.Get("Authorization")); userID != "" {
					span.SetAttributes(attribute.String("user.id", userID))
				}
				if route := chi.RouteContext(r.Context()).RoutePattern(); route != "" {
					span.SetAttributes(attribute.String("http.route", route))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func TraceParent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

func TraceIDs(ctx context.Context) (string, string) {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return "", ""
	}
	return spanContext.TraceID().String(), spanContext.SpanID().String()
}

func ContextWithTraceParent(ctx context.Context, traceparent string) context.Context {
	if traceparent == "" {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": traceparent})
}

func StartConsumerSpan(ctx context.Context, serviceName, topic, correlationID, tenantID string) (context.Context, trace.Span) {
	ctx, span := otel.Tracer(serviceName).Start(ctx, "consume "+topic)
	span.SetAttributes(
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination", topic),
		attribute.String("correlation.id", correlationID),
		attribute.String("tenant.id", tenantID),
		attribute.String("service.name", serviceName),
	)
	return ctx, span
}

func sampler() sdktrace.Sampler {
	ratio, err := strconv.ParseFloat(os.Getenv("OTEL_TRACES_SAMPLER_ARG"), 64)
	if err != nil || ratio < 0 || ratio > 1 {
		ratio = 1
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
}

func jwtSubject(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(header, prefix)), ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Subject string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Subject
}
