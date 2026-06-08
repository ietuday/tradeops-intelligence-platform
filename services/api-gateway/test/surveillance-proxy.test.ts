import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway surveillance proxy', () => {
  const originalSurveillanceServiceUrl = process.env.SURVEILLANCE_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.SURVEILLANCE_SERVICE_URL = 'http://surveillance-service.test:8090';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.SURVEILLANCE_SERVICE_URL = originalSurveillanceServiceUrl;
  });

  it('forwards /api/surveillance/health to surveillance-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'surveillance-service' }));

    const response = await request(createApp()).get('/api/surveillance/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('surveillance-service');
    expect(fetchMock).toHaveBeenCalledWith('http://surveillance-service.test:8090/health', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards alert list filters and Authorization header', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { alerts: [], limit: 25, offset: 0 }));

    const response = await request(createApp())
      .get('/api/surveillance/alerts?status=OPEN&limit=25')
      .set('authorization', 'Bearer token')
      .set('x-correlation-id', 'corr-1');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://surveillance-service.test:8090/api/v1/surveillance/alerts?status=OPEN&limit=25', expect.objectContaining({ method: 'GET' }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['x-correlation-id']).toBe('corr-1');
  });

  it('forwards alert lookup and lifecycle routes only for UUID paths', async () => {
    const id = '98e3d0cd-68d3-4a25-b4ff-7f04a329731e';
    fetchMock
      .mockResolvedValueOnce(jsonResponse(200, { id }))
      .mockResolvedValueOnce(jsonResponse(200, { id, status: 'ACKNOWLEDGED' }))
      .mockResolvedValueOnce(jsonResponse(200, { id, status: 'RESOLVED' }))
      .mockResolvedValueOnce(jsonResponse(200, { id, status: 'DISMISSED' }));

    await request(createApp()).get(`/api/surveillance/alerts/${id}`);
    await request(createApp()).post(`/api/surveillance/alerts/${id}/acknowledge`);
    await request(createApp()).post(`/api/surveillance/alerts/${id}/resolve`);
    await request(createApp()).post(`/api/surveillance/alerts/${id}/dismiss`);

    expect(fetchMock).toHaveBeenNthCalledWith(1, `http://surveillance-service.test:8090/api/v1/surveillance/alerts/${id}`, expect.objectContaining({ method: 'GET' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, `http://surveillance-service.test:8090/api/v1/surveillance/alerts/${id}/acknowledge`, expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(3, `http://surveillance-service.test:8090/api/v1/surveillance/alerts/${id}/resolve`, expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(4, `http://surveillance-service.test:8090/api/v1/surveillance/alerts/${id}/dismiss`, expect.objectContaining({ method: 'POST' }));
  });

  it('does not forward unsupported surveillance routes', async () => {
    const response = await request(createApp()).post('/api/surveillance/alerts');

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
