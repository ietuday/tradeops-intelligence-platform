import { Router, Request, Response, NextFunction } from 'express';
import { tenantHeaders } from '../middleware/tenant';
import { fetchUpstream } from './proxy-utils';

const DEFAULT_NOTIFICATION_SERVICE_URL = 'http://notification-service:8091';
const UUID_PATTERN = '[0-9a-fA-F-]{36}';

export function notificationsProxyRouter(
  notificationServiceUrl = process.env.NOTIFICATION_SERVICE_URL || DEFAULT_NOTIFICATION_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(notificationServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const notificationPath = resolveNotificationPath(req.method, req.path);
      if (!notificationPath) {
        res.status(404).json({
          error: {
            code: 'NOTIFICATION_ROUTE_NOT_FOUND',
            message: `Notification route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }
      await forwardToNotificationService(req, res, baseUrl, notificationPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(notificationServiceUrl: string): URL {
  const parsedUrl = new URL(notificationServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('NOTIFICATION_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolveNotificationPath(method: string, requestPath: string): string | undefined {
  const normalizedMethod = method.toUpperCase();
  if (normalizedMethod === 'GET' && requestPath === '/health') {
    return '/health';
  }
  if (normalizedMethod === 'GET' && requestPath === '/ready') {
    return '/ready';
  }
  if (normalizedMethod === 'GET' && requestPath === '/metrics') {
    return '/metrics';
  }
  if (normalizedMethod === 'GET' && requestPath === '/') {
    return '/api/v1/notifications';
  }
  if (normalizedMethod === 'GET' && requestPath === '/summary') {
    return '/api/v1/notifications/summary';
  }
  if (normalizedMethod === 'GET' && requestPath === '/preferences') {
    return '/api/v1/notifications/preferences';
  }
  if (normalizedMethod === 'PUT' && requestPath === '/preferences') {
    return '/api/v1/notifications/preferences';
  }

  const notificationMatch = new RegExp(`^/(${UUID_PATTERN})$`).exec(requestPath);
  if (normalizedMethod === 'GET' && notificationMatch) {
    return `/api/v1/notifications/${notificationMatch[1]}`;
  }

  const transitionMatch = new RegExp(`^/(${UUID_PATTERN})/(mark-read|retry)$`).exec(requestPath);
  if (normalizedMethod === 'POST' && transitionMatch) {
    return `/api/v1/notifications/${transitionMatch[1]}/${transitionMatch[2]}`;
  }

  return undefined;
}

function buildProxyHeaders(req: Request): HeadersInit {
  const headers: Record<string, string> = {
    accept: 'application/json'
  };
  const contentType = req.header('content-type');
  const authorization = req.header('authorization');
  const correlationId = req.header('x-correlation-id');

  if (contentType) {
    headers['content-type'] = contentType;
  }
  if (authorization) {
    headers.authorization = authorization;
  }
  if (correlationId) {
    headers['x-correlation-id'] = correlationId;
  }
  return { ...headers, ...tenantHeaders(req) };
}

function shouldForwardBody(method: string): boolean {
  return !['GET', 'HEAD'].includes(method.toUpperCase());
}

async function forwardToNotificationService(
  req: Request,
  res: Response,
  baseUrl: URL,
  notificationPath: string
): Promise<void> {
  const upstreamUrl = new URL(notificationPath, baseUrl);
  upstreamUrl.search = new URL(req.url, 'http://gateway.local').search;

  const upstream = await fetchUpstream('notification-service', upstreamUrl.toString(), {
    method: req.method,
    headers: buildProxyHeaders(req),
    body: shouldForwardBody(req.method) ? JSON.stringify(req.body ?? {}) : undefined
  });

  const responseText = await upstream.text();
  if (!responseText) {
    res.status(upstream.status).end();
    return;
  }

  const contentType = upstream.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    try {
      const jsonBody = JSON.parse(responseText) as unknown;
      res.status(upstream.status).type('application/json').json(jsonBody);
    } catch {
      res.status(502).json({
        error: {
          code: 'INVALID_UPSTREAM_RESPONSE',
          message: 'Notification service returned an invalid JSON response.'
        }
      });
    }
    return;
  }

  if (contentType.includes('text/plain')) {
    res.status(upstream.status).type('text/plain').send(responseText);
    return;
  }

  res.status(502).json({
    error: {
      code: 'UNSUPPORTED_UPSTREAM_RESPONSE',
      message: 'Notification service returned an unsupported response type.'
    }
  });
}
