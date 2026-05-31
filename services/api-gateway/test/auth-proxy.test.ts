import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway identity auth proxy', () => {
  const originalIdentityServiceUrl = process.env.IDENTITY_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.IDENTITY_SERVICE_URL = 'http://identity-service.test:8080';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.IDENTITY_SERVICE_URL = originalIdentityServiceUrl;
  });

  it('forwards /api/auth/health to identity-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {
      status: 'ok',
      service: 'identity-service'
    }));

    const response = await request(createApp())
      .get('/api/auth/health')
      .set('x-correlation-id', 'test-correlation-id');

    expect(response.status).toBe(200);
    expect(response.body).toEqual({
      status: 'ok',
      service: 'identity-service'
    });
    expect(fetchMock).toHaveBeenCalledWith('http://identity-service.test:8080/health', expect.objectContaining({
      method: 'GET'
    }));
  });

  it('forwards /api/auth/ready to identity-service /ready', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {
      status: 'ready',
      service: 'identity-service'
    }));

    const response = await request(createApp())
      .get('/api/auth/ready')
      .set('x-correlation-id', 'test-correlation-id');

    expect(response.status).toBe(200);
    expect(response.body).toEqual({
      status: 'ready',
      service: 'identity-service'
    });
    expect(fetchMock).toHaveBeenCalledWith('http://identity-service.test:8080/ready', expect.objectContaining({
      method: 'GET'
    }));
  });

  it('forwards POST /api/auth/login to identity-service /auth/login', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      tokenType: 'Bearer',
      expiresIn: 900
    }));

    const response = await request(createApp())
      .post('/api/auth/login')
      .set('authorization', 'Bearer existing-token')
      .set('content-type', 'application/json')
      .set('x-correlation-id', 'test-correlation-id')
      .send({
        email: 'trader@example.com',
        password: 'Password@123'
      });

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Headers;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://identity-service.test:8080/auth/login', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({
        email: 'trader@example.com',
        password: 'Password@123'
      })
    }));
    expect(headers.get('authorization')).toBe('Bearer existing-token');
    expect(headers.get('content-type')).toMatch(/^application\/json/);
    expect(headers.get('x-correlation-id')).toBe('test-correlation-id');
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
