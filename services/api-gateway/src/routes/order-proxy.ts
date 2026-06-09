import { Router, Request, Response, NextFunction } from 'express';
import { tenantHeaders } from '../middleware/tenant';
import { fetchUpstream, withTraceHeaders } from './proxy-utils';

const DEFAULT_ORDER_SERVICE_URL = 'http://order-service:8080';
const UUID_PATTERN = '[0-9a-fA-F-]{36}';

export function orderProxyRouter(
  orderServiceUrl = process.env.ORDER_SERVICE_URL || DEFAULT_ORDER_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(orderServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const orderPath = resolveOrderPath(req.method, req.path);
      if (!orderPath) {
        res.status(404).json({
          error: {
            code: 'ORDER_ROUTE_NOT_FOUND',
            message: `Order route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }

      await forwardToOrderService(req, res, baseUrl, orderPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(orderServiceUrl: string): URL {
  const parsedUrl = new URL(orderServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('ORDER_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolveOrderPath(method: string, requestPath: string): string | undefined {
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
  if (requestPath === '/') {
    if (normalizedMethod === 'POST') {
      return '/orders';
    }
    if (normalizedMethod === 'GET') {
      return '/orders';
    }
  }

  const orderMatch = new RegExp(`^/(${UUID_PATTERN})$`).exec(requestPath);
  if (normalizedMethod === 'GET' && orderMatch) {
    return `/orders/${orderMatch[1]}`;
  }

  const cancelMatch = new RegExp(`^/(${UUID_PATTERN})/cancel$`).exec(requestPath);
  if (normalizedMethod === 'POST' && cancelMatch) {
    return `/orders/${cancelMatch[1]}/cancel`;
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
  const idempotencyKey = req.header('idempotency-key');

  if (contentType) {
    headers['content-type'] = contentType;
  }
  if (authorization) {
    headers.authorization = authorization;
  }
  if (correlationId) {
    headers['x-correlation-id'] = correlationId;
  }
  if (idempotencyKey) {
    headers['idempotency-key'] = idempotencyKey;
  }
  return withTraceHeaders({ ...headers, ...tenantHeaders(req) }, req);
}

function shouldForwardBody(method: string): boolean {
  return !['GET', 'HEAD'].includes(method.toUpperCase());
}

async function forwardToOrderService(
  req: Request,
  res: Response,
  baseUrl: URL,
  orderPath: string
): Promise<void> {
  const upstream = await fetchUpstream('order-service', new URL(orderPath, baseUrl).toString(), {
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
          message: 'Order service returned an invalid JSON response.'
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
      message: 'Order service returned an unsupported response type.'
    }
  });
}
