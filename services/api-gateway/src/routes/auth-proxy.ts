import { Router, Request, Response, NextFunction } from 'express';

const DEFAULT_IDENTITY_SERVICE_URL = 'http://identity-service:8080';

const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade'
]);

type RouteKey = `${string} ${string}`;

const IDENTITY_ROUTE_MAP: Record<RouteKey, string> = {
  'GET /health': '/health',
  'GET /ready': '/ready',
  'GET /metrics': '/metrics',

  'POST /register': '/auth/register',
  'POST /login': '/auth/login',
  'POST /refresh': '/auth/refresh',
  'POST /logout': '/auth/logout',
  'GET /me': '/auth/me'
};

function normalizeBaseUrl(identityServiceUrl: string): URL {
  const parsedUrl = new URL(identityServiceUrl);

  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('IDENTITY_SERVICE_URL must use http or https');
  }

  return parsedUrl;
}

function resolveIdentityPath(method: string, requestPath: string): string | undefined {
  const routeKey = `${method.toUpperCase()} ${requestPath}` as RouteKey;
  return IDENTITY_ROUTE_MAP[routeKey];
}

function buildTargetUrl(baseUrl: URL, fixedIdentityPath: string): string {
  const targetUrl = new URL(fixedIdentityPath, baseUrl);
  return targetUrl.toString();
}

function shouldForwardBody(method: string): boolean {
  return !['GET', 'HEAD'].includes(method.toUpperCase());
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

  return headers;
}

async function forwardToIdentityService(
  req: Request,
  res: Response,
  baseUrl: URL,
  identityPath: string
): Promise<void> {
  const targetUrl = buildTargetUrl(baseUrl, identityPath);

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

  const responseBody = await upstream.text();
  res.send(responseBody);
}

export function authProxyRouter(
  identityServiceUrl = process.env.IDENTITY_SERVICE_URL || DEFAULT_IDENTITY_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(identityServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const identityPath = resolveIdentityPath(req.method, req.path);

      if (!identityPath) {
        res.status(404).json({
          error: {
            code: 'AUTH_ROUTE_NOT_FOUND',
            message: `Auth route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }

      await forwardToIdentityService(req, res, baseUrl, identityPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}