import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway health endpoints', () => {
  const app = createApp();

  it('returns health status', async () => {
    const response = await request(app).get('/health');

    expect(response.status).toBe(200);
    expect(response.body).toEqual({
      status: 'ok',
      service: 'api-gateway'
    });
    expect(response.headers['x-correlation-id']).toBeDefined();
  });

  it('returns readiness status', async () => {
    const response = await request(app).get('/ready');

    expect(response.status).toBe(200);
    expect(response.body).toEqual({
      status: 'ready',
      service: 'api-gateway'
    });
  });
});