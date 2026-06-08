import request from 'supertest';
import { createApp } from '../src/index';

describe('API Gateway notifications proxy', () => {
  const originalNotificationServiceUrl = process.env.NOTIFICATION_SERVICE_URL;
  const fetchMock = jest.fn();

  beforeEach(() => {
    process.env.NOTIFICATION_SERVICE_URL = 'http://notification-service.test:8091';
    fetchMock.mockReset();
    global.fetch = fetchMock;
  });

  afterAll(() => {
    process.env.NOTIFICATION_SERVICE_URL = originalNotificationServiceUrl;
  });

  it('forwards /api/notifications/health to notification-service /health', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { status: 'ok', service: 'notification-service' }));

    const response = await request(createApp()).get('/api/notifications/health');

    expect(response.status).toBe(200);
    expect(response.body.service).toBe('notification-service');
    expect(fetchMock).toHaveBeenCalledWith('http://notification-service.test:8091/health', expect.objectContaining({ method: 'GET' }));
  });

  it('forwards notification list filters and Authorization header', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { notifications: [], limit: 25, offset: 0 }));

    const response = await request(createApp())
      .get('/api/notifications?status=SENT&limit=25')
      .set('authorization', 'Bearer token')
      .set('x-correlation-id', 'corr-1');

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://notification-service.test:8091/api/v1/notifications?status=SENT&limit=25', expect.objectContaining({ method: 'GET' }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['x-correlation-id']).toBe('corr-1');
  });

  it('forwards preferences with request body', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { inAppEnabled: true, minPriority: 'HIGH' }));

    const response = await request(createApp())
      .put('/api/notifications/preferences')
      .set('authorization', 'Bearer token')
      .set('content-type', 'application/json')
      .send({ inAppEnabled: true, webhookEnabled: false, emailEnabled: false, minPriority: 'HIGH' });

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Record<string, string>;

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith('http://notification-service.test:8091/api/v1/notifications/preferences', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ inAppEnabled: true, webhookEnabled: false, emailEnabled: false, minPriority: 'HIGH' })
    }));
    expect(headers.authorization).toBe('Bearer token');
    expect(headers['content-type']).toMatch(/^application\/json/);
  });

  it('forwards notification lookup and lifecycle routes only for UUID paths', async () => {
    const id = '98e3d0cd-68d3-4a25-b4ff-7f04a329731e';
    fetchMock
      .mockResolvedValueOnce(jsonResponse(200, { id }))
      .mockResolvedValueOnce(jsonResponse(200, { id, status: 'READ' }))
      .mockResolvedValueOnce(jsonResponse(200, { id, status: 'RETRYING' }));

    await request(createApp()).get(`/api/notifications/${id}`);
    await request(createApp()).post(`/api/notifications/${id}/mark-read`);
    await request(createApp()).post(`/api/notifications/${id}/retry`);

    expect(fetchMock).toHaveBeenNthCalledWith(1, `http://notification-service.test:8091/api/v1/notifications/${id}`, expect.objectContaining({ method: 'GET' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, `http://notification-service.test:8091/api/v1/notifications/${id}/mark-read`, expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(3, `http://notification-service.test:8091/api/v1/notifications/${id}/retry`, expect.objectContaining({ method: 'POST' }));
  });

  it('does not forward unsupported notification routes', async () => {
    const response = await request(createApp()).post('/api/notifications');

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
