import { dlqCatalog, serviceRegistry, topicCatalog } from '../config/serviceRegistry';
import { recordAdminHealthCheck } from '../observability/metrics';
import { traceContextHeaders } from '../observability/tracing';
import { AdminContext, AdminStatus, DownstreamResult, ServiceRegistryEntry } from '../types/admin';

const DEFAULT_HEALTH_TIMEOUT_MS = 1500;
const PLATFORM_VERSION = '2.7.0';

export class AdminService {
  services() {
    return { services: serviceRegistry().map(stripInternalRegistryFields) };
  }

  topics() {
    return { topics: topicCatalog };
  }

  dlqSummary() {
    return { dlqs: dlqCatalog };
  }

  opsChecklist() {
    return {
      checklists: [
        { name: 'Production Readiness', doc: 'docs/production-readiness/checklist.md' },
        { name: 'Observability Runbook', doc: 'docs/observability/runbook.md' },
        { name: 'DLQ Runbook', doc: 'docs/reliability/dead-letter-topics.md' },
        { name: 'OpenTelemetry Runbook', doc: 'docs/tracing/otel-runbook.md' },
        { name: 'Tenant Isolation', doc: 'docs/multitenancy/tenant-isolation.md' },
        { name: 'Rule Configuration', doc: 'docs/surveillance/rule-configuration.md' }
      ]
    };
  }

  platformConfig() {
    return {
      environment: process.env.NODE_ENV || process.env.ENVIRONMENT || 'local',
      version: PLATFORM_VERSION,
      features: {
        multiTenancy: true,
        webSockets: (process.env.WS_ENABLED || 'true').toLowerCase() === 'true',
        openTelemetry: (process.env.OTEL_ENABLED || 'false').toLowerCase() === 'true',
        ruleConfiguration: true,
        eventSchemaGovernance: true,
        adminApi: (process.env.ADMIN_API_ENABLED || 'true').toLowerCase() === 'true'
      },
      infrastructure: {
        postgres: configured(process.env.DATABASE_URL || process.env.POSTGRES_DSN),
        redpanda: configured(process.env.KAFKA_BROKERS || process.env.REDPANDA_BROKERS),
        redis: configured(process.env.REDIS_URL),
        jaeger: configured(process.env.OTEL_EXPORTER_OTLP_ENDPOINT)
      },
      safeConfig: {
        adminHealthTimeoutMs: healthTimeoutMs(),
        adminServiceRegistryMode: process.env.ADMIN_SERVICE_REGISTRY_MODE || 'static',
        identityServiceUrl: maskUrl(process.env.IDENTITY_SERVICE_URL),
        marketDataServiceUrl: maskUrl(process.env.MARKET_DATA_SERVICE_URL),
        orderServiceUrl: maskUrl(process.env.ORDER_SERVICE_URL),
        portfolioServiceUrl: maskUrl(process.env.PORTFOLIO_SERVICE_URL),
        strategyServiceUrl: maskUrl(process.env.STRATEGY_SERVICE_URL),
        riskServiceUrl: maskUrl(process.env.RISK_SERVICE_URL),
        surveillanceServiceUrl: maskUrl(process.env.SURVEILLANCE_SERVICE_URL),
        notificationServiceUrl: maskUrl(process.env.NOTIFICATION_SERVICE_URL),
        auditServiceUrl: maskUrl(process.env.AUDIT_SERVICE_URL),
        databaseUrl: maskUrl(process.env.DATABASE_URL),
        redisUrl: maskUrl(process.env.REDIS_URL)
      }
    };
  }

  async healthSummary() {
    const checkedAt = new Date().toISOString();
    const checks = await Promise.allSettled(serviceRegistry().map((entry) => this.checkService(entry)));
    const services = checks.map((result, index) => {
      if (result.status === 'fulfilled') {
        return result.value;
      }
      const entry = serviceRegistry()[index];
      return failedCheck(entry, 'Health check failed.');
    });
    const failed = services.filter((service) => service.status !== 'HEALTHY');
    const criticalFailed = failed.some((service) => serviceRegistry().find((entry) => entry.name === service.name)?.critical);

    return {
      status: failed.length === 0 ? 'HEALTHY' : criticalFailed ? 'UNHEALTHY' : 'DEGRADED',
      checkedAt,
      services
    };
  }

  async auditSummary(ctx: AdminContext, query: URLSearchParams) {
    const result = await getJson<Record<string, unknown>>('audit-service', process.env.AUDIT_SERVICE_URL || 'http://audit-service:8092', '/api/v1/audit/summary', ctx, query);
    return withTenantAndDegraded(ctx.tenantId, result, {
      window: windowFromQuery(query),
      summary: { totalEvents: 0, securityEvents: 0, orderEvents: 0, ruleConfigEvents: 0 },
      recent: []
    });
  }

  async alertsSummary(ctx: AdminContext, query: URLSearchParams) {
    const result = await getJson<Record<string, unknown>>('surveillance-service', process.env.SURVEILLANCE_SERVICE_URL || 'http://surveillance-service:8090', '/api/v1/surveillance/alerts/summary', ctx, query);
    return withTenantAndDegraded(ctx.tenantId, result, {
      summary: { open: 0, acknowledged: 0, resolvedToday: 0, critical: 0, high: 0 },
      recent: []
    });
  }

  async notificationsSummary(ctx: AdminContext, query: URLSearchParams) {
    const result = await getJson<Record<string, unknown>>('notification-service', process.env.NOTIFICATION_SERVICE_URL || 'http://notification-service:8091', '/api/v1/notifications/summary', ctx, query);
    return withTenantAndDegraded(ctx.tenantId, result, {
      summary: { unread: 0, failed: 0, sentToday: 0, pendingRetries: 0 },
      recentFailures: []
    });
  }

  async ruleConfigSummary(ctx: AdminContext, query: URLSearchParams) {
    const result = await getJson<{ rules?: unknown[] }>('surveillance-service', process.env.SURVEILLANCE_SERVICE_URL || 'http://surveillance-service:8090', '/api/v1/surveillance/rules', ctx, query);
    if (result.status === 'DEGRADED') {
      return withTenantAndDegraded(ctx.tenantId, result, {
        rules: { total: 0, enabled: 0, disabled: 0, critical: 0, high: 0 },
        disabledRules: []
      });
    }

    const rules = Array.isArray(result.data.rules) ? result.data.rules : [];
    const summary = rules.reduce<{
      total: number;
      enabled: number;
      disabled: number;
      critical: number;
      high: number;
      disabledRules: string[];
    }>((acc, rule) => {
      const item = rule as { enabled?: unknown; severity?: unknown; ruleName?: unknown; name?: unknown };
      const enabled = item.enabled !== false;
      const severity = typeof item.severity === 'string' ? item.severity.toUpperCase() : '';
      acc.total += 1;
      acc.enabled += enabled ? 1 : 0;
      acc.disabled += enabled ? 0 : 1;
      acc.critical += severity === 'CRITICAL' ? 1 : 0;
      acc.high += severity === 'HIGH' ? 1 : 0;
      if (!enabled) {
        acc.disabledRules.push(String(item.ruleName || item.name || 'unknown-rule'));
      }
      return acc;
    }, { total: 0, enabled: 0, disabled: 0, critical: 0, high: 0, disabledRules: [] as string[] });

    return {
      tenantId: ctx.tenantId,
      status: 'OK',
      rules: {
        total: summary.total,
        enabled: summary.enabled,
        disabled: summary.disabled,
        critical: summary.critical,
        high: summary.high
      },
      disabledRules: summary.disabledRules
    };
  }

  cacheRefresh() {
    return {
      status: 'ACCEPTED',
      message: 'Manual cache refresh is not configured for this deployment.'
    };
  }

  private async checkService(entry: ServiceRegistryEntry) {
    if (entry.name === 'api-gateway') {
      const result = {
        name: entry.name,
        status: 'HEALTHY' as const,
        healthUrl: new URL(entry.healthPath, entry.baseUrl).toString(),
        latencyMs: 0,
        error: null
      };
      recordAdminHealthCheck(entry.name, result.status, result.latencyMs);
      return result;
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), healthTimeoutMs());
    const healthUrl = new URL(entry.healthPath, entry.baseUrl).toString();
    const startedAt = Date.now();
    try {
      const response = await fetch(healthUrl, {
        method: 'GET',
        headers: { accept: 'application/json', ...traceContextHeaders() },
        signal: controller.signal
      });
      const latencyMs = Date.now() - startedAt;
      const status = response.ok ? 'HEALTHY' : 'UNHEALTHY';
      recordAdminHealthCheck(entry.name, status, latencyMs);
      return {
        name: entry.name,
        status,
        healthUrl,
        latencyMs,
        error: response.ok ? null : `HTTP ${response.status}`
      };
    } catch (error) {
      const latencyMs = Date.now() - startedAt;
      const message = error instanceof Error && error.name === 'AbortError' ? 'Health check timed out.' : 'Health check failed.';
      recordAdminHealthCheck(entry.name, 'UNHEALTHY', latencyMs);
      return {
        name: entry.name,
        status: 'UNHEALTHY' as const,
        healthUrl,
        latencyMs,
        error: message
      };
    } finally {
      clearTimeout(timeout);
    }
  }
}

function stripInternalRegistryFields(entry: ServiceRegistryEntry) {
  const { critical: _critical, ...publicEntry } = entry;
  return publicEntry;
}

function failedCheck(entry: ServiceRegistryEntry, error: string) {
  const result = {
    name: entry.name,
    status: 'UNHEALTHY' as const,
    healthUrl: new URL(entry.healthPath, entry.baseUrl).toString(),
    latencyMs: 0,
    error
  };
  recordAdminHealthCheck(entry.name, result.status, result.latencyMs);
  return result;
}

function healthTimeoutMs(): number {
  const configured = Number(process.env.ADMIN_HEALTH_TIMEOUT_MS || DEFAULT_HEALTH_TIMEOUT_MS);
  return Number.isFinite(configured) && configured > 0 ? configured : DEFAULT_HEALTH_TIMEOUT_MS;
}

async function getJson<T>(service: string, baseUrl: string, path: string, ctx: AdminContext, query: URLSearchParams): Promise<DownstreamResult<T>> {
  const url = new URL(path, baseUrl);
  query.forEach((value, key) => {
    if (key !== 'tenantId') {
      url.searchParams.append(key, value);
    }
  });

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), healthTimeoutMs());
  try {
    const response = await fetch(url.toString(), {
      method: 'GET',
      headers: {
        accept: 'application/json',
        authorization: ctx.authorization,
        'x-tenant-id': ctx.tenantId,
        ...(ctx.correlationId ? { 'x-correlation-id': ctx.correlationId } : {}),
        ...traceContextHeaders()
      },
      signal: controller.signal
    });
    if (!response.ok) {
      return { status: 'DEGRADED', data: {} as T, error: `${service} returned HTTP ${response.status}.` };
    }
    return { status: 'OK', data: await response.json() as T };
  } catch (error) {
    const message = error instanceof Error && error.name === 'AbortError' ? `${service} timed out.` : `${service} is unavailable.`;
    return { status: 'DEGRADED', data: {} as T, error: message };
  } finally {
    clearTimeout(timeout);
  }
}

function withTenantAndDegraded<T extends Record<string, unknown>>(tenantId: string, result: DownstreamResult<Record<string, unknown>>, fallback: T) {
  if (result.status === 'OK') {
    return { tenantId, status: 'OK', ...result.data };
  }
  return {
    tenantId,
    status: 'DEGRADED' as AdminStatus,
    ...fallback,
    error: result.error || 'Downstream summary is unavailable.'
  };
}

function windowFromQuery(query: URLSearchParams) {
  return {
    from: query.get('from') || null,
    to: query.get('to') || null
  };
}

function configured(value?: string): string {
  return value ? 'configured' : 'not_configured';
}

export function maskUrl(value?: string): string | undefined {
  if (!value) {
    return undefined;
  }
  try {
    const url = new URL(value);
    if (url.password) {
      url.password = '****';
    }
    if (url.username && ['postgres:', 'postgresql:', 'redis:'].includes(url.protocol)) {
      return url.toString();
    }
    return url.toString();
  } catch {
    return value.replace(/\/\/([^:/\s]+):([^@\s]+)@/g, '//$1:****@');
  }
}
