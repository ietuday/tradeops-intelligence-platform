import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway risk proxy', () => {
  const originalRiskServiceUrl = process.env.RISK_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.RISK_SERVICE_URL = 'http://risk-engine-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.RISK_SERVICE_URL = originalRiskServiceUrl;
  });

  it('forwards /api/risk/health to risk-engine-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'risk-engine-service' }));

    const response = await request(createApp()).get('/api/risk/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('risk-engine-service');
    expect(fetchMock).toHaveBeenCalledWith('http://risk-engine-service.test:8080/health', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards /api/risk/ready to risk-engine-service /ready', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ready', service: 'risk-engine-service' }));

    const response = await request(createApp()).get('/api/risk/ready');

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://risk-engine-service.test:8080/ready', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards risk routes with Authorization header', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { score: 21, level: 'LOW' }));

    const response = await request(createApp())
      .get('/api/risk/portfolio/score')
      .set('authorization', 'Bearer token')
      .set('x-correlation-id', 'corr-1');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://risk-engine-service.test:8080/risk/portfolio/score', expect.objectContaining({ method: 'GET' }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['x-correlation-id']).toBe('corr-1');
  });

  it('forwards advanced risk analytics routes with body and tenant headers', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { portfolioId: 'demo', scenarioResults: [] }));

    const body = {
      portfolioId: 'demo',
      positions: [{ symbol: 'AAPL', quantity: 10, averagePrice: 150, currentPrice: 180 }],
      scenarios: [{ name: 'Market drops 10%', marketShockPercent: -10 }]
    };

    const response = await request(createApp())
      .post('/api/risk/stress-test')
      .set('authorization', bearerToken({ sub: 'risk-1', tenantId: 'tenant-a', roles: ['risk_manager'] }))
      .set('x-correlation-id', 'corr-1')
      .send(body);

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://risk-engine-service.test:8080/api/v1/risk/stress-test', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify(body)
    }));
    expect(headers['x-tenant-id']).toBe('tenant-a');
    expect(headers['x-correlation-id']).toBe('corr-1');
  });

  it('forwards portfolio-specific advanced risk analytics routes', async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse(200, { portfolioId: 'demo-portfolio-1' }))
      .mockResolvedValueOnce(jsonResponse(200, { portfolioId: 'demo-portfolio-1' }));

    await request(createApp()).get('/api/risk/portfolio/demo-portfolio-1/concentration');
    await request(createApp()).get('/api/risk/portfolio/demo-portfolio-1/drawdown-trend?limit=20');

    expect(fetchMock).toHaveBeenNthCalledWith(1, 'http://risk-engine-service.test:8080/api/v1/risk/portfolio/demo-portfolio-1/concentration', expect.objectContaining({ method: 'GET' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, 'http://risk-engine-service.test:8080/api/v1/risk/portfolio/demo-portfolio-1/drawdown-trend?limit=20', expect.objectContaining({ method: 'GET' }));
  });

  it('does not forward unsupported risk routes', async () => {
    const response = await request(createApp()).post('/api/risk/portfolio/score');

    expect(response.status).toBe(404);
    expect(fetchMock).not.toHaveBeenCalled();
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

function bearerToken(payload: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'none', typ: 'JWT' })).toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  return `Bearer ${header}.${body}.signature`;
}
