import { Router, Request, Response, NextFunction } from 'express';

const DEFAULT_RISK_SERVICE_URL = 'http://risk-engine-service:8080';

type RouteKey = `${string} ${string}`;

const RISK_ROUTE_MAP: Record<RouteKey, string> = {
  'GET /health': '/health',
  'GET /ready': '/ready',
  'GET /metrics': '/metrics',
  'GET /portfolio/score': '/risk/portfolio/score',
  'GET /portfolio/volatility': '/risk/portfolio/volatility',
  'GET /portfolio/drawdown': '/risk/portfolio/drawdown',
  'GET /portfolio/var': '/risk/portfolio/var',
  'GET /recommendations': '/risk/recommendations',
  'GET /anomalies': '/risk/anomalies'
};

export function riskProxyRouter(
  riskServiceUrl = process.env.RISK_SERVICE_URL || DEFAULT_RISK_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(riskServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const riskPath = resolveRiskPath(req.method, req.path);
      if (!riskPath) {
        res.status(404).json({
          error: {
            code: 'RISK_ROUTE_NOT_FOUND',
            message: `Risk route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }
      await forwardToRiskService(req, res, baseUrl, riskPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(riskServiceUrl: string): URL {
  const parsedUrl = new URL(riskServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('RISK_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolveRiskPath(method: string, requestPath: string): string | undefined {
  const routeKey = `${method.toUpperCase()} ${requestPath}` as RouteKey;
  return RISK_ROUTE_MAP[routeKey];
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

async function forwardToRiskService(
  req: Request,
  res: Response,
  baseUrl: URL,
  riskPath: string
): Promise<void> {
  const upstream = await fetch(new URL(riskPath, baseUrl).toString(), {
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
          message: 'Risk engine service returned an invalid JSON response.'
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
      message: 'Risk engine service returned an unsupported response type.'
    }
  });
}
