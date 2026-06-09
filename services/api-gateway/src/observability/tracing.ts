import { context, propagation, trace } from '@opentelemetry/api';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { NodeSDK } from '@opentelemetry/sdk-node';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { Request, Response, NextFunction } from 'express';
import { parsePrincipal, tenantHeaders } from '../middleware/tenant';

let sdk: NodeSDK | undefined;

export function startTracing(): void {
  if ((process.env.OTEL_ENABLED || 'false').toLowerCase() !== 'true') {
    return;
  }

  try {
    const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://localhost:4318';
    sdk = new NodeSDK({
      resource: resourceFromAttributes({
        [ATTR_SERVICE_NAME]: 'api-gateway',
        [ATTR_SERVICE_VERSION]: process.env.OTEL_SERVICE_VERSION || process.env.npm_package_version || '0.1.0',
        'tradeops.service': 'api-gateway'
      }),
      traceExporter: new OTLPTraceExporter({ url: normalizeOtlpHttpEndpoint(endpoint) }),
      instrumentations: [getNodeAutoInstrumentations()]
    });
    sdk.start();
  } catch (error) {
    // Tracing must never block local startup or tests.
    // eslint-disable-next-line no-console
    console.warn('OpenTelemetry tracing disabled after initialization failure', error);
  }
}

export async function shutdownTracing(): Promise<void> {
  if (!sdk) {
    return;
  }
  try {
    await sdk.shutdown();
  } catch {
    // Ignore exporter shutdown errors during service termination.
  }
}

export function tracingMiddleware(req: Request, _res: Response, next: NextFunction): void {
  const span = trace.getSpan(context.active());
  if (span) {
    span.setAttribute('tradeops.service', 'api-gateway');
    span.setAttribute('correlation.id', req.header('x-correlation-id') || '');
    span.setAttribute('http.route', req.route?.path?.toString() || req.path);

    const tenantId = tenantHeaders(req)['x-tenant-id'];
    if (tenantId) {
      span.setAttribute('tenant.id', tenantId);
    }
    const principal = parsePrincipal(req.header('authorization'));
    if (principal?.userId) {
      span.setAttribute('user.id', principal.userId);
    }
  }
  next();
}

export function traceContextHeaders(): Record<string, string> {
  const carrier: Record<string, string> = {};
  propagation.inject(context.active(), carrier);
  return pickTraceHeaders(carrier);
}

function pickTraceHeaders(headers: Record<string, string>): Record<string, string> {
  const picked: Record<string, string> = {};
  for (const key of ['traceparent', 'tracestate']) {
    const value = headers[key];
    if (value) {
      picked[key] = value;
    }
  }
  return picked;
}

function normalizeOtlpHttpEndpoint(endpoint: string): string {
  return endpoint.endsWith('/v1/traces') ? endpoint : `${endpoint.replace(/\/$/, '')}/v1/traces`;
}

startTracing();
