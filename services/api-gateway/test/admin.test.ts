import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway admin operations APIs', () => {
  const fetchMock = jest.fn();
  const originalEnv = { ...process.env };

  beforeEach(() => {
    process.env = { ...originalEnv };
    process.env.ADMIN_API_ENABLED = 'true';
    process.env.ADMIN_HEALTH_TIMEOUT_MS = '50';
    process.env.IDENTITY_SERVICE_URL = 'http://identity-service.test:8080';
    process.env.MARKET_DATA_SERVICE_URL = 'http://market-data-service.test:8080';
    process.env.ORDER_SERVICE_URL = 'http://order-service.test:8080';
    process.env.PORTFOLIO_SERVICE_URL = 'http://portfolio-service.test:8080';
    process.env.STRATEGY_SERVICE_URL = 'http://strategy-service.test:8080';
    process.env.RISK_SERVICE_URL = 'http://risk-engine-service.test:8080';
    process.env.SURVEILLANCE_SERVICE_URL = 'http://surveillance-service.test:8090';
    process.env.NOTIFICATION_SERVICE_URL = 'http://notification-service.test:8091';
    process.env.AUDIT_SERVICE_URL = 'http://audit-service.test:8092';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  it('rejects unauthenticated and non-admin callers', async () => {
    const unauthenticated = await request(createApp()).get('/api/admin/services');
    const forbidden = await request(createApp())
      .get('/api/admin/services')
      .set('authorization', bearerToken({ sub: 'user-1', tenantId: 'tenant-a', roles: ['trader'] }));

    expect(unauthenticated.status).toBe(401);
    expect(forbidden.status).toBe(403);
  });

  it('allows trading_admin and risk_manager on read-only endpoints', async () => {
    const admin = await request(createApp())
      .get('/api/admin/services')
      .set('authorization', bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] }));
    const riskManager = await request(createApp())
      .get('/api/admin/topics')
      .set('authorization', bearerToken({ sub: 'risk-1', tenantId: 'tenant-a', roles: ['risk_manager'] }));

    expect(admin.status).toBe(200);
    expect(admin.body.services).toEqual(expect.arrayContaining([expect.objectContaining({ name: 'order-service' })]));
    expect(riskManager.status).toBe(200);
    expect(riskManager.body.topics).toEqual(expect.arrayContaining([expect.objectContaining({ topic: 'order.created' })]));
  });

  it('restricts mutating admin actions to trading_admin', async () => {
    const response = await request(createApp())
      .post('/api/admin/cache/refresh')
      .set('authorization', bearerToken({ sub: 'risk-1', tenantId: 'tenant-a', roles: ['risk_manager'] }));

    expect(response.status).toBe(403);
  });

  it('returns HEALTHY when all downstream health checks pass', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok' }));

    const response = await request(createApp())
      .get('/api/admin/health-summary')
      .set('authorization', bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] }));

    expect(response.status).toBe(200);
    expect(response.body.status).toBe('HEALTHY');
    expect(response.body.services).toHaveLength(10);
    expect(response.body.services[0]).toEqual(expect.objectContaining({ name: 'api-gateway', status: 'HEALTHY' }));
  });

  it('returns DEGRADED when a non-critical service fails', async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url.includes('notification-service')) {
        return Promise.resolve(jsonResponse(500, { status: 'down' }));
      }
      return Promise.resolve(jsonResponse(200, { status: 'ok' }));
    });

    const response = await request(createApp())
      .get('/api/admin/health-summary')
      .set('authorization', bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] }));

    expect(response.status).toBe(200);
    expect(response.body.status).toBe('DEGRADED');
    expect(response.body.services).toEqual(expect.arrayContaining([
      expect.objectContaining({ name: 'notification-service', status: 'UNHEALTHY', error: 'HTTP 500' })
    ]));
  });

  it('returns UNHEALTHY when a critical service fails', async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url.includes('order-service')) {
        return Promise.resolve(jsonResponse(503, { status: 'down' }));
      }
      return Promise.resolve(jsonResponse(200, { status: 'ok' }));
    });

    const response = await request(createApp())
      .get('/api/admin/health-summary')
      .set('authorization', bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] }));

    expect(response.body.status).toBe('UNHEALTHY');
  });

  it('returns known DLQs and topic schema paths', async () => {
    const token = bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] });
    const dlqs = await request(createApp()).get('/api/admin/dlq-summary').set('authorization', token);
    const topics = await request(createApp()).get('/api/admin/topics').set('authorization', token);

    expect(dlqs.body.dlqs).toEqual(expect.arrayContaining([
      expect.objectContaining({ topic: 'surveillance.dlq', runbook: 'docs/reliability/dead-letter-topics.md' })
    ]));
    expect(topics.body.topics).toEqual(expect.arrayContaining([
      expect.objectContaining({ topic: 'order.created', schema: 'schemas/events/orders/order.created.v1.json' })
    ]));
  });

  it('aggregates rule config summary from surveillance rules', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {
      rules: [
        { ruleName: 'LargeOrderRule', enabled: true, severity: 'HIGH' },
        { ruleName: 'AbnormalPriceMovementRule', enabled: false, severity: 'CRITICAL' }
      ]
    }));

    const response = await request(createApp())
      .get('/api/admin/rule-config-summary?tenantId=tenant-b')
      .set('authorization', bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] }));

    const [, init] = fetchMock.mock.calls[0];
    expect((init.headers as Record<string, string>)['x-tenant-id']).toBe('tenant-b');
    expect(response.body).toEqual(expect.objectContaining({
      tenantId: 'tenant-b',
      status: 'OK',
      rules: { total: 2, enabled: 1, disabled: 1, critical: 1, high: 1 },
      disabledRules: ['AbnormalPriceMovementRule']
    }));
  });

  it('degrades summary endpoints when downstream service fails', async () => {
    fetchMock.mockRejectedValue(new Error('network down'));

    const response = await request(createApp())
      .get('/api/admin/alerts-summary')
      .set('authorization', bearerToken({ sub: 'risk-1', tenantId: 'tenant-a', roles: ['risk_manager'] }));

    expect(response.status).toBe(200);
    expect(response.body.status).toBe('DEGRADED');
    expect(response.body.tenantId).toBe('tenant-a');
    expect(response.body.summary).toEqual({ open: 0, acknowledged: 0, resolvedToday: 0, critical: 0, high: 0 });
  });

  it('masks sensitive platform config values', async () => {
    process.env.DATABASE_URL = 'postgres://tradeops:secret@postgres:5432/tradeops';
    process.env.REDIS_URL = 'redis://:redis-secret@redis:6379/0';

    const response = await request(createApp())
      .get('/api/admin/platform-config')
      .set('authorization', bearerToken({ sub: 'admin-1', tenantId: 'tenant-a', roles: ['trading_admin'] }));

    const serialized = JSON.stringify(response.body);
    expect(response.body.version).toBe('2.7.0');
    expect(response.body.safeConfig.databaseUrl).toContain('****');
    expect(serialized).not.toContain('secret@');
    expect(serialized).not.toContain('redis-secret');
  });
});

function bearerToken(payload: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'none', typ: 'JWT' })).toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  return `Bearer ${header}.${body}.signature`;
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type': 'application/json'
    }
  });
}
