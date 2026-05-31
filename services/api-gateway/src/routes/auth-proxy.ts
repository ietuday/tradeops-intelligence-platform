import { Router, Request, Response, NextFunction } from 'express';
import { CORRELATION_ID_HEADER } from '../middleware/correlation-id';

const DEFAULT_IDENTITY_SERVICE_URL = 'http://identity-service:8080';
const DIRECT_IDENTITY_PATHS = new Set(['/health', '/ready', '/metrics']);
const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'content-length',
  'host',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade'
]);

export function authProxyRouter(identityServiceUrl = process.env.IDENTITY_SERVICE_URL || DEFAULT_IDENTITY_SERVICE_URL) {
  const router = Router();
  const baseUrl = identityServiceUrl.replace(/\/$/, '');

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const targetUrl = mapIdentityTargetUrl(baseUrl, req.url);
      const upstream = await fetch(targetUrl, {
        method: req.method,
        headers: buildProxyHeaders(req),
        body: shouldForwardBody(req.method) ? JSON.stringify(req.body ?? {}) : undefined
      });

      res.status(upstream.status);
      upstream.headers.forEach((value, key) => {
        if (!HOP_BY_HOP_HEADERS.has(key.toLowerCase())) {
          res.setHeader(key, value);
        }
      });
      res.setHeader(CORRELATION_ID_HEADER, req.headers[CORRELATION_ID_HEADER] as string);
      res.send(Buffer.from(await upstream.arrayBuffer()));
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function mapIdentityTargetUrl(baseUrl: string, mountedUrl: string): string {
  const incoming = new URL(mountedUrl, baseUrl);
  const targetPath = DIRECT_IDENTITY_PATHS.has(incoming.pathname)
    ? incoming.pathname
    : `/auth${incoming.pathname}`;
  const target = new URL(targetPath, baseUrl);
  target.search = incoming.search;
  return target.toString();
}

function buildProxyHeaders(req: Request): Headers {
  const headers = new Headers();
  for (const [key, value] of Object.entries(req.headers)) {
    if (value === undefined || HOP_BY_HOP_HEADERS.has(key.toLowerCase())) {
      continue;
    }
    headers.set(key, Array.isArray(value) ? value.join(',') : value);
  }
  headers.set(CORRELATION_ID_HEADER, req.headers[CORRELATION_ID_HEADER] as string);
  return headers;
}

function shouldForwardBody(method: string): boolean {
  return !['GET', 'HEAD'].includes(method);
}
