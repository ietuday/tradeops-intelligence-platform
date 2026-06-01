import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway portfolio proxy', () => {
  const originalPortfolioServiceUrl = process.env.PORTFOLIO_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.PORTFOLIO_SERVICE_URL = 'http://portfolio-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.PORTFOLIO_SERVICE_URL = originalPortfolioServiceUrl;
  });

  it('forwards /api/portfolio/health to portfolio-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'portfolio-service' }));

    const response = await request(createApp()).get('/api/portfolio/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('portfolio-service');
    expect(fetchMock).toHaveBeenCalledWith('http://portfolio-service.test:8080/health', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards portfolio data routes with Authorization header', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { holdings: [] }));

    const response = await request(createApp())
      .get('/api/portfolio/holdings')
      .set('authorization', 'Bearer token');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://portfolio-service.test:8080/portfolio/holdings', expect.objectContaining({ method: 'GET' }));
    expect(headers.authorization).toBe('Bearer token');
  });

  it('forwards /api/portfolio/metrics as text', async () => {
    fetchMock.mockResolvedValue(new Response('portfolio_updates_total 1\n', {
      status: 200,
      headers: { 'content-type': 'text/plain; version=0.0.4' }
    }));

    const response = await request(createApp()).get('/api/portfolio/metrics');

    expect(response.status).toBe(200);
    expect(response.text).toContain('portfolio_updates_total');
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
