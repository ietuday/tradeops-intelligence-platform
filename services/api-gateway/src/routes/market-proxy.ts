import { Router, Request, Response, NextFunction } from 'express';

const DEFAULT_MARKET_DATA_SERVICE_URL = 'http://market-data-service:8080';

type RouteKey = `${string} ${string}`;

const MARKET_ROUTE_MAP: Record<RouteKey, string> = {
  'GET /health': '/health',
  'GET /ready': '/ready',
  'GET /metrics': '/metrics',
  'GET /ticks/latest': '/ticks/latest',
  'GET /symbols': '/symbols'
};

export function marketProxyRouter(
  marketDataServiceUrl = process.env.MARKET_DATA_SERVICE_URL || DEFAULT_MARKET_DATA_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(marketDataServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const marketPath = resolveMarketPath(req.method, req.path);
      if (!marketPath) {
        res.status(404).json({
          error: {
            code: 'MARKET_ROUTE_NOT_FOUND',
            message: `Market route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }

      await forwardToMarketDataService(req, res, baseUrl, marketPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(marketDataServiceUrl: string): URL {
  const parsedUrl = new URL(marketDataServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('MARKET_DATA_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolveMarketPath(method: string, requestPath: string): string | undefined {
  const routeKey = `${method.toUpperCase()} ${requestPath}` as RouteKey;
  return MARKET_ROUTE_MAP[routeKey];
}

function buildTargetUrl(baseUrl: URL, fixedMarketPath: string): string {
  return new URL(fixedMarketPath, baseUrl).toString();
}

function buildProxyHeaders(req: Request): HeadersInit {
  const headers: Record<string, string> = {};
  const correlationId = req.header('x-correlation-id');
  if (correlationId) {
    headers['x-correlation-id'] = correlationId;
  }
  return headers;
}

async function forwardToMarketDataService(
  req: Request,
  res: Response,
  baseUrl: URL,
  marketPath: string
): Promise<void> {
  const upstream = await fetch(buildTargetUrl(baseUrl, marketPath), {
    method: req.method,
    headers: buildProxyHeaders(req)
  });

  const body = await upstream.text();
  const contentType = upstream.headers.get('content-type') || '';

  if (!body) {
    res.status(upstream.status).end();
    return;
  }

  if (contentType.includes('application/json')) {
    try {
      res.status(upstream.status).type('application/json').json(JSON.parse(body));
    } catch {
      res.status(502).json({
        error: {
          code: 'INVALID_UPSTREAM_RESPONSE',
          message: 'Market data service returned an invalid JSON response.'
        }
      });
    }
    return;
  }

  if (contentType.includes('text/plain')) {
    res.status(upstream.status).type('text/plain').send(body);
    return;
  }

  res.status(502).json({
    error: {
      code: 'UNSUPPORTED_UPSTREAM_RESPONSE',
      message: 'Market data service returned an unsupported response type.'
    }
  });
}
