import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway strategy proxy', () => {
  const originalStrategyServiceUrl = process.env.STRATEGY_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.STRATEGY_SERVICE_URL = 'http://strategy-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.STRATEGY_SERVICE_URL = originalStrategyServiceUrl;
  });

  it('forwards /api/strategies/health to strategy-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'strategy-service' }));

    const response = await request(createApp()).get('/api/strategies/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('strategy-service');
    expect(fetchMock).toHaveBeenCalledWith('http://strategy-service.test:8080/health', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards /api/strategies/ready to strategy-service /ready', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ready', service: 'strategy-service' }));

    const response = await request(createApp()).get('/api/strategies/ready');

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://strategy-service.test:8080/ready', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards POST /api/strategies with request body and headers', async () => {
    fetchMock.mockResolvedValue(jsonResponse(201, { id: 'strategy-1' }));

    const response = await request(createApp())
      .post('/api/strategies')
      .set('authorization', 'Bearer token')
      .set('content-type', 'application/json')
      .set('x-correlation-id', 'corr-1')
      .send({ name: 'MA Cross' });

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(201);
    expect(fetchMock).toHaveBeenCalledWith('http://strategy-service.test:8080/strategies', expect.objectContaining({ method: 'POST' }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['content-type']).toContain('application/json');
    expect(headers['x-correlation-id']).toBe('corr-1');
    expect(init.body).toBe(JSON.stringify({ name: 'MA Cross' }));
  });

  it('forwards strategy id child routes through explicit allowlist', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { totalReturn: 1 }));

    const response = await request(createApp()).get('/api/strategies/11111111-1111-1111-1111-111111111111/performance');

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith(
      'http://strategy-service.test:8080/strategies/11111111-1111-1111-1111-111111111111/performance',
      expect.objectContaining({ method: 'GET' })
    );
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
