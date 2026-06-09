import { Server as HttpServer, IncomingMessage } from 'http';
import pino from 'pino';
import { Duplex } from 'stream';
import { WebSocket, WebSocketServer } from 'ws';
import { v4 as uuidv4 } from 'uuid';
import {
  recordWebSocketAuthFailure,
  recordWebSocketConnectionClosed,
  recordWebSocketConnectionOpened
} from '../observability/metrics';
import { extractToken, verifyJwt } from './auth';
import { WebSocketEventHub } from './event-hub';
import { canAccessStream, resolveStream, WebSocketStream } from './streams';

export interface WebSocketServerConfig {
  requireAuth: boolean;
  allowedOrigins: string[];
  maxConnections: number;
  heartbeatIntervalMs: number;
}

export interface AttachedWebSocketServer {
  hub: WebSocketEventHub;
  server: WebSocketServer;
  close: () => void;
}

export function attachWebSocketServer(
  httpServer: HttpServer,
  logger: pino.Logger,
  config = loadWebSocketServerConfig(),
  hub = new WebSocketEventHub()
): AttachedWebSocketServer {
  const wsServer = new WebSocketServer({ noServer: true });

  httpServer.on('upgrade', (req, socket, head) => {
    const host = req.headers.host || 'localhost';
    const url = new URL(req.url || '/', `http://${host}`);
    const stream = resolveStream(url.pathname);
    const correlationId = req.headers['x-correlation-id']?.toString() || url.searchParams.get('correlationId') || uuidv4();

    if (!stream) {
      rejectUpgrade(socket, 404, 'WebSocket route not found');
      return;
    }

    if (!originAllowed(req, config.allowedOrigins)) {
      recordWebSocketAuthFailure(stream);
      rejectUpgrade(socket, 403, 'WebSocket origin not allowed');
      return;
    }

    if (hub.subscriberCount() >= config.maxConnections) {
      rejectUpgrade(socket, 429, 'Too many WebSocket connections');
      return;
    }

    if (config.requireAuth) {
      try {
        const token = extractToken(req, url);
        if (!token) {
          throw new Error('missing token');
        }
        const principal = verifyJwt(token);
        if (!canAccessStream(principal.roles, stream)) {
          throw new Error('role not allowed for stream');
        }
      } catch (error) {
        recordWebSocketAuthFailure(stream);
        logger.warn({ err: error, stream, correlationId }, 'WebSocket authentication failed');
        rejectUpgrade(socket, 401, 'Unauthorized');
        return;
      }
    }

    wsServer.handleUpgrade(req, socket, head, (ws) => {
      wsServer.emit('connection', ws, req, stream, correlationId);
    });
  });

  wsServer.on('connection', (socket: WebSocket, _req: IncomingMessage, stream: WebSocketStream, correlationId: string) => {
    logger.info({ stream, correlationId }, 'WebSocket client connected');
    recordWebSocketConnectionOpened(stream);
    const unsubscribe = hub.subscribe(socket, stream);

    sendJson(socket, {
      type: 'connection.ready',
      stream,
      correlationId,
      timestamp: new Date().toISOString()
    });

    const heartbeat = setInterval(() => {
      if (socket.readyState !== WebSocket.OPEN) {
        return;
      }
      sendJson(socket, {
        type: 'heartbeat',
        stream,
        correlationId,
        timestamp: new Date().toISOString()
      });
      socket.ping();
    }, config.heartbeatIntervalMs);

    socket.on('close', () => {
      clearInterval(heartbeat);
      unsubscribe();
      recordWebSocketConnectionClosed(stream);
      logger.info({ stream, correlationId }, 'WebSocket client disconnected');
    });

    socket.on('error', (error) => {
      logger.warn({ err: error, stream, correlationId }, 'WebSocket client error');
    });
  });

  return {
    hub,
    server: wsServer,
    close: () => wsServer.close()
  };
}

export function loadWebSocketServerConfig(): WebSocketServerConfig {
  return {
    requireAuth: (process.env.WS_REQUIRE_AUTH || 'true').toLowerCase() === 'true',
    allowedOrigins: (process.env.WS_ALLOWED_ORIGINS || 'http://localhost:4200,http://localhost:4300')
      .split(',')
      .map((origin) => origin.trim())
      .filter((origin) => origin.length > 0),
    maxConnections: positiveNumber(process.env.WS_MAX_CONNECTIONS, 100),
    heartbeatIntervalMs: positiveNumber(process.env.WS_HEARTBEAT_INTERVAL_MS, 30000)
  };
}

function originAllowed(req: IncomingMessage, allowedOrigins: string[]): boolean {
  const origin = req.headers.origin;
  if (!origin) {
    return true;
  }
  return allowedOrigins.includes(origin);
}

function rejectUpgrade(socket: Duplex, status: number, reason: string): void {
  socket.write(`HTTP/1.1 ${status} ${reason}\r\nConnection: close\r\nContent-Type: text/plain\r\n\r\n${reason}`);
  socket.destroy();
}

function sendJson(socket: WebSocket, value: unknown): void {
  if (socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(value));
  }
}

function positiveNumber(value: string | undefined, fallback: number): number {
  const parsed = Number(value || fallback);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}
