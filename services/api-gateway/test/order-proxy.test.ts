import request from 'supertest';
import crypto from 'crypto';
import { createApp } from '../src/index';

describe('API Gateway order proxy', () => {
  const originalOrderServiceUrl = process.env.ORDER_SERVICE_URL;
  const originalProxyTimeoutMs = process.env.PROXY_TIMEOUT_MS;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.ORDER_SERVICE_URL = 'http://order-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.ORDER_SERVICE_URL = originalOrderServiceUrl;
    process.env.PROXY_TIMEOUT_MS = originalProxyTimeoutMs;
  });

  it('forwards /api/orders/health to order-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'order-service' }));

    const response = await request(createApp()).get('/api/orders/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('order-service');
    expect(fetchMock).toHaveBeenCalledWith('http://order-service.test:8080/health', expect.objectContaining({
      method: 'GET'
    }));
  });

  it('forwards POST /api/orders to order-service /orders', async () => {
    fetchMock.mockResolvedValue(jsonResponse(201, { id: 'order-1', status: 'filled' }));

    const response = await request(createApp())
      .post('/api/orders')
      .set('authorization', 'Bearer token')
      .set('content-type', 'application/json')
      .set('idempotency-key', 'idem-1')
      .send({ symbol: 'AAPL', side: 'BUY', orderType: 'MARKET', quantity: 10 });

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(201);
    expect(fetchMock).toHaveBeenCalledWith('http://order-service.test:8080/orders', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ symbol: 'AAPL', side: 'BUY', orderType: 'MARKET', quantity: 10 })
    }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['idempotency-key']).toBe('idem-1');
    expect(headers['content-type']).toMatch(/^application\/json/);
    expect(headers['x-correlation-id']).toBeDefined();
  });

  it('forwards X-Tenant-ID from JWT tenantId', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { orders: [] }));
    const token = signToken('tenant-a', ['trader']);

    await request(createApp())
      .get('/api/orders')
      .set('authorization', `Bearer ${token}`);

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;
    expect(headers['x-tenant-id']).toBe('tenant-a');
  });

  it('does not let external X-Tenant-ID override JWT tenantId for non-admins', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { orders: [] }));
    const token = signToken('tenant-a', ['trader']);

    await request(createApp())
      .get('/api/orders')
      .set('authorization', `Bearer ${token}`)
      .set('x-tenant-id', 'tenant-b');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;
    expect(headers['x-tenant-id']).toBe('tenant-a');
  });

  it('allows trading_admin to override X-Tenant-ID for tenant support workflows', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { orders: [] }));
    const token = signToken('tenant-a', ['trading_admin']);

    await request(createApp())
      .get('/api/orders')
      .set('authorization', `Bearer ${token}`)
      .set('x-tenant-id', 'tenant-b');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;
    expect(headers['x-tenant-id']).toBe('tenant-b');
  });

  it('forwards order lookup and cancel routes only for UUID paths', async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse(200, { id: '98e3d0cd-68d3-4a25-b4ff-7f04a329731e' }))
      .mockResolvedValueOnce(jsonResponse(200, { id: '98e3d0cd-68d3-4a25-b4ff-7f04a329731e', status: 'cancelled' }));

    await request(createApp()).get('/api/orders/98e3d0cd-68d3-4a25-b4ff-7f04a329731e');
    await request(createApp()).post('/api/orders/98e3d0cd-68d3-4a25-b4ff-7f04a329731e/cancel');

    expect(fetchMock).toHaveBeenNthCalledWith(1, 'http://order-service.test:8080/orders/98e3d0cd-68d3-4a25-b4ff-7f04a329731e', expect.objectContaining({ method: 'GET' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, 'http://order-service.test:8080/orders/98e3d0cd-68d3-4a25-b4ff-7f04a329731e/cancel', expect.objectContaining({ method: 'POST' }));
  });

  it('returns 502 when upstream order-service is unavailable', async () => {
    fetchMock.mockRejectedValue(new Error('connection refused'));

    const response = await request(createApp())
      .get('/api/orders/health')
      .set('x-correlation-id', 'order-corr-502');

    expect(response.status).toBe(502);
    expect(response.headers['x-correlation-id']).toBe('order-corr-502');
    expect(response.body).toEqual({
      error: {
        code: 'UPSTREAM_UNAVAILABLE',
        message: 'order-service is unavailable.',
        correlationId: 'order-corr-502'
      }
    });
  });

  it('returns 504 when upstream order-service times out', async () => {
    process.env.PROXY_TIMEOUT_MS = '10';
    fetchMock.mockImplementation((_url, init: RequestInit) => new Promise((_resolve, reject) => {
      const signal = init.signal as AbortSignal;
      signal.addEventListener('abort', () => {
        const error = new Error('aborted');
        error.name = 'AbortError';
        reject(error);
      });
    }));

    const response = await request(createApp())
      .get('/api/orders/health')
      .set('x-correlation-id', 'order-corr-504');

    expect(response.status).toBe(504);
    expect(response.headers['x-correlation-id']).toBe('order-corr-504');
    expect(response.body).toEqual({
      error: {
        code: 'UPSTREAM_TIMEOUT',
        message: 'order-service did not respond before the proxy timeout.',
        correlationId: 'order-corr-504'
      }
    });
  });
});

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type': 'application/json'
    }
  });
}

function signToken(tenantId: string, roles: string[]): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    sub: 'user-1',
    tenantId,
    email: 'demo@example.com',
    roles,
    iss: 'tradeops-identity-service',
    exp: Math.floor(Date.now() / 1000) + 3600
  })).toString('base64url');
  const signature = crypto
    .createHmac('sha256', 'local_dev_jwt_secret_change_me_123456789')
    .update(`${header}.${payload}`)
    .digest('base64url');
  return `${header}.${payload}.${signature}`;
}
