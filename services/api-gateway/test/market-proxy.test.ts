import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway market data proxy', () => {
  const originalMarketDataServiceUrl = process.env.MARKET_DATA_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.MARKET_DATA_SERVICE_URL = 'http://market-data-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.MARKET_DATA_SERVICE_URL = originalMarketDataServiceUrl;
  });

  it('forwards /api/market/health to market-data-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {
      status: 'ok',
      service: 'market-data-service'
    }));

    const response = await request(createApp()).get('/api/market/health');

    expect(response.status).toBe(200);
    expect(response.body).toEqual({
      status: 'ok',
      service: 'market-data-service'
    });
    expect(fetchMock).toHaveBeenCalledWith('http://market-data-service.test:8080/health', expect.objectContaining({
      method: 'GET'
    }));
  });

  it('forwards /api/market/ready to market-data-service /ready', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {
      status: 'ready',
      service: 'market-data-service'
    }));

    const response = await request(createApp()).get('/api/market/ready');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('market-data-service');
    expect(fetchMock).toHaveBeenCalledWith('http://market-data-service.test:8080/ready', expect.objectContaining({
      method: 'GET'
    }));
  });

  it('forwards /api/market/metrics to market-data-service /metrics', async () => {
    fetchMock.mockResolvedValue(new Response('market_ticks_received_total 1\n', {
      status: 200,
      headers: {
        'content-type': 'text/plain; version=0.0.4'
      }
    }));

    const response = await request(createApp()).get('/api/market/metrics');

    expect(response.status).toBe(200);
    expect(response.text).toContain('market_ticks_received_total');
    expect(fetchMock).toHaveBeenCalledWith('http://market-data-service.test:8080/metrics', expect.objectContaining({
      method: 'GET'
    }));
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
