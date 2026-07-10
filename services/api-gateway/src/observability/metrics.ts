import { Request, Response, NextFunction } from 'express';
import client from 'prom-client';

const register = new client.Registry();

client.collectDefaultMetrics({
  register,
  prefix: 'tradeops_api_gateway_'
});

const httpRequestsTotal = new client.Counter({
  name: 'tradeops_api_gateway_http_requests_total',
  help: 'Total number of HTTP requests received by the API Gateway.',
  labelNames: ['method', 'route', 'status_code'],
  registers: [register]
});

const httpRequestDurationSeconds = new client.Histogram({
  name: 'tradeops_api_gateway_http_request_duration_seconds',
  help: 'HTTP request duration in seconds for the API Gateway.',
  labelNames: ['method', 'route', 'status_code'],
  buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5],
  registers: [register]
});

const proxyUpstreamErrorsTotal = new client.Counter({
  name: 'tradeops_api_gateway_proxy_upstream_errors_total',
  help: 'Total upstream proxy errors by service and status.',
  labelNames: ['service', 'status'],
  registers: [register]
});

const proxyUpstreamTimeoutsTotal = new client.Counter({
  name: 'tradeops_api_gateway_proxy_upstream_timeouts_total',
  help: 'Total upstream proxy timeouts by service.',
  labelNames: ['service'],
  registers: [register]
});

const websocketConnectionsActive = new client.Gauge({
  name: 'tradeops_api_gateway_websocket_connections_active',
  help: 'Active WebSocket connections by stream.',
  labelNames: ['stream'],
  registers: [register]
});

const websocketConnectionsTotal = new client.Counter({
  name: 'tradeops_api_gateway_websocket_connections_total',
  help: 'Total WebSocket connections accepted by stream.',
  labelNames: ['stream'],
  registers: [register]
});

const websocketMessagesSentTotal = new client.Counter({
  name: 'tradeops_api_gateway_websocket_messages_sent_total',
  help: 'Total WebSocket messages sent by stream and topic.',
  labelNames: ['stream', 'topic'],
  registers: [register]
});

const websocketMessagesFailedTotal = new client.Counter({
  name: 'tradeops_api_gateway_websocket_messages_failed_total',
  help: 'Total WebSocket message send failures by stream and topic.',
  labelNames: ['stream', 'topic'],
  registers: [register]
});

const websocketAuthFailuresTotal = new client.Counter({
  name: 'tradeops_api_gateway_websocket_auth_failures_total',
  help: 'Total WebSocket authentication failures by stream.',
  labelNames: ['stream'],
  registers: [register]
});

const websocketKafkaEventsConsumedTotal = new client.Counter({
  name: 'tradeops_api_gateway_websocket_kafka_events_consumed_total',
  help: 'Total Kafka events consumed for WebSocket streaming by topic.',
  labelNames: ['topic'],
  registers: [register]
});

const adminRequestsTotal = new client.Counter({
  name: 'tradeops_api_gateway_admin_requests_total',
  help: 'Total admin API requests by endpoint and status.',
  labelNames: ['endpoint', 'status'],
  registers: [register]
});

const adminHealthChecksTotal = new client.Counter({
  name: 'tradeops_api_gateway_admin_health_checks_total',
  help: 'Total admin health checks by service and status.',
  labelNames: ['service', 'status'],
  registers: [register]
});

const adminHealthCheckDurationMs = new client.Histogram({
  name: 'tradeops_api_gateway_admin_health_check_duration_ms',
  help: 'Admin health check duration in milliseconds by service.',
  labelNames: ['service'],
  buckets: [5, 10, 25, 50, 100, 250, 500, 1000, 1500, 3000, 5000],
  registers: [register]
});

const authRequestsTotal = new client.Counter({
  name: 'tradeops_auth_requests_total',
  help: 'Total authentication decisions by bounded reason.',
  labelNames: ['service', 'auth_type', 'decision', 'reason'],
  registers: [register]
});

const authTokenValidationTotal = new client.Counter({
  name: 'tradeops_auth_token_validation_total',
  help: 'Total JWT validation attempts by status and bounded reason.',
  labelNames: ['service', 'status', 'reason'],
  registers: [register]
});

const authForbiddenTotal = new client.Counter({
  name: 'tradeops_auth_forbidden_total',
  help: 'Total authorization failures by route group and bounded reason.',
  labelNames: ['service', 'route_group', 'reason'],
  registers: [register]
});

const authTenantMismatchTotal = new client.Counter({
  name: 'tradeops_auth_tenant_mismatch_total',
  help: 'Total tenant mismatch authorization failures by route group.',
  labelNames: ['service', 'route_group'],
  registers: [register]
});

const authJwksRefreshTotal = new client.Counter({
  name: 'tradeops_auth_jwks_refresh_total',
  help: 'Total JWKS refresh attempts by status.',
  labelNames: ['service', 'status'],
  registers: [register]
});

const authJwksCacheAgeSeconds = new client.Gauge({
  name: 'tradeops_auth_jwks_cache_age_seconds',
  help: 'Current JWKS cache age in seconds.',
  labelNames: ['service'],
  registers: [register]
});

const serviceAuthRequestsTotal = new client.Counter({
  name: 'tradeops_service_auth_requests_total',
  help: 'Total service auth decisions by caller and bounded reason.',
  labelNames: ['service', 'caller', 'decision', 'reason'],
  registers: [register]
});

function normalizeRoute(req: Request): string {
  return req.route?.path?.toString() || req.path || 'unknown';
}

export function metricsMiddleware(req: Request, res: Response, next: NextFunction): void {
  const start = process.hrtime.bigint();

  res.on('finish', () => {
    const durationSeconds = Number(process.hrtime.bigint() - start) / 1_000_000_000;
    const labels = {
      method: req.method,
      route: normalizeRoute(req),
      status_code: String(res.statusCode)
    };

    httpRequestsTotal.inc(labels);
    httpRequestDurationSeconds.observe(labels, durationSeconds);
  });

  next();
}

export async function metricsHandler(_req: Request, res: Response): Promise<void> {
  res.set('Content-Type', register.contentType);
  res.end(await register.metrics());
}

export function recordProxyUpstreamError(service: string, status: number): void {
  proxyUpstreamErrorsTotal.inc({ service, status: String(status) });
}

export function recordProxyUpstreamTimeout(service: string): void {
  proxyUpstreamTimeoutsTotal.inc({ service });
}

export function recordWebSocketConnectionOpened(stream: string): void {
  websocketConnectionsActive.inc({ stream });
  websocketConnectionsTotal.inc({ stream });
}

export function recordWebSocketConnectionClosed(stream: string): void {
  websocketConnectionsActive.dec({ stream });
}

export function recordWebSocketMessageSent(stream: string, topic: string): void {
  websocketMessagesSentTotal.inc({ stream, topic });
}

export function recordWebSocketMessageFailed(stream: string, topic: string): void {
  websocketMessagesFailedTotal.inc({ stream, topic });
}

export function recordWebSocketAuthFailure(stream: string): void {
  websocketAuthFailuresTotal.inc({ stream });
}

export function recordWebSocketKafkaEventConsumed(topic: string): void {
  websocketKafkaEventsConsumedTotal.inc({ topic });
}

export function recordAdminRequest(endpoint: string, status: number): void {
  adminRequestsTotal.inc({ endpoint, status: String(status) });
}

export function recordAdminHealthCheck(service: string, status: string, durationMs: number): void {
  adminHealthChecksTotal.inc({ service, status });
  adminHealthCheckDurationMs.observe({ service }, durationMs);
}

export function recordAuthRequest(authType: string, decision: string, reason: string): void {
  authRequestsTotal.inc({ service: 'api-gateway', auth_type: authType, decision, reason: normalizeReason(reason) });
}

export function recordAuthTokenValidation(status: string, reason: string): void {
  authTokenValidationTotal.inc({ service: 'api-gateway', status, reason: normalizeReason(reason) });
}

export function recordForbidden(routeGroup: string, reason: string): void {
  authForbiddenTotal.inc({ service: 'api-gateway', route_group: routeGroup, reason: normalizeReason(reason) });
}

export function recordTenantMismatch(routeGroup: string): void {
  authTenantMismatchTotal.inc({ service: 'api-gateway', route_group: routeGroup });
}

export function recordJwksRefresh(status: string): void {
  authJwksRefreshTotal.inc({ service: 'api-gateway', status: normalizeReason(status) });
}

export function setJwksCacheAge(ageSeconds: number): void {
  authJwksCacheAgeSeconds.set({ service: 'api-gateway' }, ageSeconds);
}

export function recordServiceAuth(caller: string, decision: string, reason: string): void {
  serviceAuthRequestsTotal.inc({ service: 'api-gateway', caller: caller || 'unknown', decision, reason: normalizeReason(reason) });
}

function normalizeReason(reason: string): string {
  return /^[a-z_]+$/.test(reason) ? reason : 'other';
}

export { register };
