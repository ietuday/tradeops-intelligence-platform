import crypto from 'crypto';
import { createServer, Server } from 'http';
import WebSocket from 'ws';
import { createApp } from '../src/index';
import { WebSocketEventHub } from '../src/realtime/event-hub';
import { attachWebSocketServer } from '../src/realtime/websocket-server';
import { topicsForStream } from '../src/realtime/streams';
import { register } from '../src/observability/metrics';
import pino from 'pino';

const logger = pino({ enabled: false });

describe('API Gateway WebSocket streaming', () => {
  let server: Server;
  let port: number;
  let hub: WebSocketEventHub;
  let sockets: WebSocket[] = [];

  afterEach(async () => {
    for (const socket of sockets) {
      socket.terminate();
    }
    sockets = [];

    if (server?.listening) {
      await new Promise<void>((resolve) => {
        server.close(() => resolve());
      });
    }
  });

  async function startGateway(requireAuth = true): Promise<void> {
    server = createServer(createApp());
    const attached = attachWebSocketServer(server, logger, {
      requireAuth,
      allowedOrigins: ['http://localhost:4200'],
      maxConnections: 10,
      heartbeatIntervalMs: 10000
    });
    hub = attached.hub;
    await new Promise<void>((resolve) => {
      server.listen(0, '127.0.0.1', () => {
        const address = server.address();
        if (address && typeof address === 'object') {
          port = address.port;
        }
        resolve();
      });
    });
  }

  it('initializes without breaking REST routes', async () => {
    await startGateway(false);

    const response = await fetch(`http://127.0.0.1:${port}/health`);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.service).toBe('api-gateway');
  });

  it('rejects a connection without token when auth is required', async () => {
    await startGateway(true);

    await expect(connect('/ws/orders')).rejects.toThrow();
  });

  it('accepts a valid token and sends connection.ready', async () => {
    await startGateway(true);
    const token = signToken(['trader']);
    const { ws, ready } = await connectAndReady(`/ws/orders?token=${token}`);

    expect(ready.type).toBe('connection.ready');
    expect(ready.stream).toBe('orders');
    expect(ready.tenantId).toBe('default-tenant');

    ws.close();
  });

  it('maps topics to stream subscriptions', async () => {
    expect(topicsForStream('orders')).toContain('order.filled');
    expect(topicsForStream('alerts')).toContain('surveillance.alert.created');
    expect(topicsForStream('notifications')).toContain('notification.created');
    expect(topicsForStream('audit')).toContain('audit.log.created');
  });

  it('routes Kafka topic events to matching stream clients', async () => {
    await startGateway(true);
    const token = signToken(['trader']);
    const { ws } = await connectAndReady(`/ws/orders?token=${token}`);

    hub.handleKafkaMessage('order.filled', JSON.stringify({ correlationId: 'ws-corr-1', orderId: 'order-1' }));
    const event = await nextMessage(ws);

    expect(event).toEqual({
      type: 'order.filled',
      topic: 'order.filled',
      correlationId: 'ws-corr-1',
      timestamp: expect.any(String),
      payload: { correlationId: 'ws-corr-1', orderId: 'order-1' }
    });

    ws.close();
  });

  it('filters WebSocket events by tenantId', async () => {
    await startGateway(true);
    const tenantAToken = signToken(['trader'], 'tenant-a');
    const tenantBToken = signToken(['trader'], 'tenant-b');
    const tenantA = await connectAndReady(`/ws/orders?token=${tenantAToken}`);
    const tenantB = await connectAndReady(`/ws/orders?token=${tenantBToken}`);

    const tenantBMessages: Record<string, unknown>[] = [];
    tenantB.ws.on('message', (data) => {
      tenantBMessages.push(JSON.parse(data.toString()) as Record<string, unknown>);
    });

    hub.handleKafkaMessage('order.filled', JSON.stringify({ tenantId: 'tenant-a', correlationId: 'ws-corr-tenant', orderId: 'order-1' }));
    const event = await nextMessage(tenantA.ws);

    expect(event.payload).toEqual({ tenantId: 'tenant-a', correlationId: 'ws-corr-tenant', orderId: 'order-1' });
    await new Promise((resolve) => setTimeout(resolve, 25));
    expect(tenantBMessages).toHaveLength(0);
  });

  it('ignores malformed Kafka payloads without throwing', async () => {
    const localHub = new WebSocketEventHub();

    expect(() => localHub.handleKafkaMessage('order.filled', '{bad json')).not.toThrow();
    expect(localHub.handleKafkaMessage('order.filled', '{bad json')).toBeUndefined();
  });

  it('exposes WebSocket metrics', async () => {
    await startGateway(true);
    const token = signToken(['trader']);
    const { ws } = await connectAndReady(`/ws/orders?token=${token}`);
    ws.close();

    const metrics = await register.metrics();
    expect(metrics).toContain('tradeops_api_gateway_websocket_connections_total');
  });

  function connect(path: string): Promise<WebSocket> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(`ws://127.0.0.1:${port}${path}`, {
        headers: {
          Origin: 'http://localhost:4200'
        }
      });
      ws.once('open', () => resolve(ws));
      ws.once('error', reject);
      ws.once('unexpected-response', (_req, res) => {
        reject(new Error(`unexpected response ${res.statusCode}`));
      });
    });
  }

  function connectAndReady(path: string): Promise<{ ws: WebSocket; ready: Record<string, unknown> }> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(`ws://127.0.0.1:${port}${path}`, {
        headers: {
          Origin: 'http://localhost:4200'
        }
      });
      sockets.push(ws);

      let settled = false;
      const fail = (error: Error) => {
        if (!settled) {
          settled = true;
          reject(error);
        }
      };

      ws.once('message', (data) => {
        if (!settled) {
          settled = true;
          resolve({ ws, ready: JSON.parse(data.toString()) as Record<string, unknown> });
        }
      });
      ws.once('error', fail);
      ws.once('unexpected-response', (_req, res) => {
        fail(new Error(`unexpected response ${res.statusCode}`));
      });
    });
  }
});

function nextMessage(ws: WebSocket): Promise<Record<string, unknown>> {
  return new Promise((resolve) => {
    ws.once('message', (data) => {
      resolve(JSON.parse(data.toString()) as Record<string, unknown>);
    });
  });
}

function signToken(roles: string[], tenantId = 'default-tenant'): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    sub: 'user-1',
    tenantId,
    email: 'demo@example.com',
    roles,
    iss: 'tradeops-identity-service',
    exp: Math.floor(Date.now() / 1000) + 3600
  })).toString('base64url');
  const signature = crypto
    .createHmac('sha256', 'local_dev_jwt_secret_change_me_123456789')
    .update(`${header}.${payload}`)
    .digest('base64url');
  return `${header}.${payload}.${signature}`;
}
