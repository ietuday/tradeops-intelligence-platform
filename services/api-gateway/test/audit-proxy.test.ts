import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway audit proxy', () => {
  const originalAuditServiceUrl = process.env.AUDIT_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.AUDIT_SERVICE_URL = 'http://audit-service.test:8092';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.AUDIT_SERVICE_URL = originalAuditServiceUrl;
  });

  it('forwards audit health checks', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'audit-service' }));

    const response = await request(createApp()).get('/api/audit/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('audit-service');
    expect(fetchMock).toHaveBeenCalledWith('http://audit-service.test:8092/health', expect.objectContaining({
      method: 'GET'
    }));
  });

  it('forwards list filters and preserves auth header', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { auditLogs: [] }));

    const response = await request(createApp())
      .get('/api/audit/logs?serviceName=order-service&limit=10')
      .set('authorization', 'Bearer token')
      .set('x-correlation-id', 'corr-1');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://audit-service.test:8092/api/v1/audit/logs?serviceName=order-service&limit=10', expect.objectContaining({ method: 'GET' }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['x-correlation-id']).toBe('corr-1');
  });

  it('returns 502 when audit-service is unavailable', async () => {
    fetchMock.mockRejectedValue(new Error('connection refused'));

    const response = await request(createApp())
      .get('/api/audit/health')
      .set('x-correlation-id', 'audit-corr-502');

    expect(response.status).toBe(502);
    expect(response.headers['x-correlation-id']).toBe('audit-corr-502');
    expect(response.body.error.code).toBe('UPSTREAM_UNAVAILABLE');
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
