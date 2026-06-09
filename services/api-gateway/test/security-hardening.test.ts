import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway security hardening', () => {
  const originalEnv = { ...process.env };

  afterEach(() => {
    process.env = { ...originalEnv };
  });

  it('sets common security headers', async () => {
    const response = await request(createApp()).get('/health');

    expect(response.status).toBe(200);
    expect(response.headers['x-dns-prefetch-control']).toBe('off');
    expect(response.headers['x-content-type-options']).toBe('nosniff');
    expect(response.headers['content-security-policy']).toBeDefined();
  });

  it('respects configured CORS origins', async () => {
    process.env.CORS_ORIGIN = 'http://localhost:4200,http://localhost:4300';

    const response = await request(createApp())
      .options('/health')
      .set('Origin', 'http://localhost:4200')
      .set('Access-Control-Request-Method', 'GET');

    expect(response.status).toBe(204);
    expect(response.headers['access-control-allow-origin']).toBe('http://localhost:4200');
  });

  it('returns 413 with a correlation ID when the request body is too large', async () => {
    process.env.REQUEST_BODY_LIMIT = '10b';

    const response = await request(createApp())
      .post('/api/orders')
      .set('Content-Type', 'application/json')
      .set('x-correlation-id', 'body-too-large-corr')
      .send({ symbol: 'AAPL', quantity: 100 });

    expect(response.status).toBe(413);
    expect(response.headers['x-correlation-id']).toBe('body-too-large-corr');
    expect(response.body).toEqual({
      error: {
        code: 'REQUEST_BODY_TOO_LARGE',
        message: 'Request body exceeds the configured size limit.',
        correlationId: 'body-too-large-corr'
      }
    });
  });

  it('returns 429 with a correlation ID after the request limit is exceeded', async () => {
    process.env.RATE_LIMIT_WINDOW_MS = '60000';
    process.env.RATE_LIMIT_MAX_REQUESTS = '1';
    const app = createApp();

    await request(app).get('/health').set('x-correlation-id', 'rate-limit-first');
    const response = await request(app)
      .get('/health')
      .set('x-correlation-id', 'rate-limit-second');

    expect(response.status).toBe(429);
    expect(response.headers['x-correlation-id']).toBe('rate-limit-second');
    expect(response.headers['retry-after']).toBeDefined();
    expect(response.body).toEqual({
      error: {
        code: 'RATE_LIMIT_EXCEEDED',
        message: 'Too many requests. Please retry after the rate limit window resets.',
        correlationId: 'rate-limit-second'
      }
    });
  });
});
