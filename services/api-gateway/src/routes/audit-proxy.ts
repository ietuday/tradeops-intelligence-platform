import { Router, Request, Response, NextFunction } from 'express';
import { tenantHeaders } from '../middleware/tenant';
import { fetchUpstream } from './proxy-utils';

const DEFAULT_AUDIT_SERVICE_URL = 'http://audit-service:8092';
const UUID_PATTERN = '[0-9a-fA-F-]{36}';

export function auditProxyRouter(
  auditServiceUrl = process.env.AUDIT_SERVICE_URL || DEFAULT_AUDIT_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(auditServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const auditPath = resolveAuditPath(req.method, req.path);
      if (!auditPath) {
        res.status(404).json({
          error: {
            code: 'AUDIT_ROUTE_NOT_FOUND',
            message: `Audit route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }
      await forwardToAuditService(req, res, baseUrl, auditPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(auditServiceUrl: string): URL {
  const parsedUrl = new URL(auditServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('AUDIT_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolveAuditPath(method: string, requestPath: string): string | undefined {
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
  if (normalizedMethod === 'GET' && requestPath === '/logs') {
    return '/api/v1/audit/logs';
  }
  if (normalizedMethod === 'GET' && requestPath === '/summary') {
    return '/api/v1/audit/summary';
  }
  if (normalizedMethod === 'GET' && requestPath === '/export') {
    return '/api/v1/audit/export';
  }
  const logMatch = new RegExp(`^/logs/(${UUID_PATTERN})$`).exec(requestPath);
  if (normalizedMethod === 'GET' && logMatch) {
    return `/api/v1/audit/logs/${logMatch[1]}`;
  }
  return undefined;
}

function buildProxyHeaders(req: Request): HeadersInit {
  const headers: Record<string, string> = {
    accept: 'application/json,text/csv,text/plain'
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

async function forwardToAuditService(
  req: Request,
  res: Response,
  baseUrl: URL,
  auditPath: string
): Promise<void> {
  const upstreamUrl = new URL(auditPath, baseUrl);
  upstreamUrl.search = new URL(req.url, 'http://gateway.local').search;

  const upstream = await fetchUpstream('audit-service', upstreamUrl.toString(), {
    method: req.method,
    headers: buildProxyHeaders(req)
  });

  const responseText = await upstream.text();
  if (!responseText) {
    res.status(upstream.status).end();
    return;
  }

  const contentType = upstream.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    try {
      res.status(upstream.status).type('application/json').json(JSON.parse(responseText) as unknown);
    } catch {
      res.status(502).json({
        error: {
          code: 'INVALID_UPSTREAM_RESPONSE',
          message: 'Audit service returned an invalid JSON response.'
        }
      });
    }
    return;
  }
  if (contentType.includes('text/csv')) {
    res.status(upstream.status).type('text/csv').send(responseText);
    return;
  }
  if (contentType.includes('text/plain')) {
    res.status(upstream.status).type('text/plain').send(responseText);
    return;
  }
  res.status(502).json({
    error: {
      code: 'UNSUPPORTED_UPSTREAM_RESPONSE',
      message: 'Audit service returned an unsupported response type.'
    }
  });
}
