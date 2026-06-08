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

export { register };
