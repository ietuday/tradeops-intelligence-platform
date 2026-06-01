import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway order proxy', () => {
  const originalOrderServiceUrl = process.env.ORDER_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.ORDER_SERVICE_URL = 'http://order-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.ORDER_SERVICE_URL = originalOrderServiceUrl;
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
});

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type': 'application/json'
    }
  });
}
