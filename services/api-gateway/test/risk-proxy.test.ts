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
