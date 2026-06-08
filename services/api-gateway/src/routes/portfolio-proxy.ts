import { Router, Request, Response, NextFunction } from 'express';
import { fetchUpstream } from './proxy-utils';

const DEFAULT_PORTFOLIO_SERVICE_URL = 'http://portfolio-service:8080';

type RouteKey = `${string} ${string}`;

const PORTFOLIO_ROUTE_MAP: Record<RouteKey, string> = {
  'GET /health': '/health',
  'GET /ready': '/ready',
  'GET /metrics': '/metrics',
  'GET /': '/portfolio',
  'GET /holdings': '/portfolio/holdings',
  'GET /snapshots': '/portfolio/snapshots',
  'GET /exposure': '/portfolio/exposure',
  'GET /pnl': '/portfolio/pnl'
};

export function portfolioProxyRouter(
  portfolioServiceUrl = process.env.PORTFOLIO_SERVICE_URL || DEFAULT_PORTFOLIO_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(portfolioServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const portfolioPath = resolvePortfolioPath(req.method, req.path);
      if (!portfolioPath) {
        res.status(404).json({
          error: {
            code: 'PORTFOLIO_ROUTE_NOT_FOUND',
            message: `Portfolio route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }
      await forwardToPortfolioService(req, res, baseUrl, portfolioPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(portfolioServiceUrl: string): URL {
  const parsedUrl = new URL(portfolioServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('PORTFOLIO_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolvePortfolioPath(method: string, requestPath: string): string | undefined {
  const routeKey = `${method.toUpperCase()} ${requestPath}` as RouteKey;
  return PORTFOLIO_ROUTE_MAP[routeKey];
}

function buildProxyHeaders(req: Request): HeadersInit {
  const headers: Record<string, string> = {
    accept: 'application/json'
  };
  const authorization = req.header('authorization');
  const correlationId = req.header('x-correlation-id');
  if (authorization) {
    headers.authorization = authorization;
  }
  if (correlationId) {
    headers['x-correlation-id'] = correlationId;
  }
  return headers;
}

async function forwardToPortfolioService(
  req: Request,
  res: Response,
  baseUrl: URL,
  portfolioPath: string
): Promise<void> {
  const upstream = await fetchUpstream('portfolio-service', new URL(portfolioPath, baseUrl).toString(), {
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
      const jsonBody = JSON.parse(responseText) as unknown;
      res.status(upstream.status).type('application/json').json(jsonBody);
    } catch {
      res.status(502).json({
        error: {
          code: 'INVALID_UPSTREAM_RESPONSE',
          message: 'Portfolio service returned an invalid JSON response.'
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
      message: 'Portfolio service returned an unsupported response type.'
    }
  });
}
