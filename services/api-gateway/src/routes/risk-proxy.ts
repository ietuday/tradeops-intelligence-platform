import { Router, Request, Response, NextFunction } from 'express';
import { tenantHeaders } from '../middleware/tenant';
import { fetchUpstream, withTraceHeaders } from './proxy-utils';

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
  'GET /anomalies': '/risk/anomalies',
  'GET /scenarios': '/api/v1/risk/scenarios',
  'POST /stress-test': '/api/v1/risk/stress-test',
  'POST /scenarios/run': '/api/v1/risk/scenarios/run',
  'POST /portfolio/concentration': '/api/v1/risk/portfolio/concentration',
  'POST /portfolio/drawdown-trend': '/api/v1/risk/portfolio/drawdown-trend',
  'POST /volatility-shock': '/api/v1/risk/volatility-shock'
};

const UUID_OR_SLUG_PATTERN = '[A-Za-z0-9._-]+';

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
  const normalizedMethod = method.toUpperCase();
  const routeKey = `${normalizedMethod} ${requestPath}` as RouteKey;
  const mapped = RISK_ROUTE_MAP[routeKey];
  if (mapped) {
    return mapped;
  }

  const concentrationMatch = new RegExp(`^/portfolio/(${UUID_OR_SLUG_PATTERN})/concentration$`).exec(requestPath);
  if (normalizedMethod === 'GET' && concentrationMatch) {
    return `/api/v1/risk/portfolio/${concentrationMatch[1]}/concentration`;
  }

  const drawdownMatch = new RegExp(`^/portfolio/(${UUID_OR_SLUG_PATTERN})/drawdown-trend$`).exec(requestPath);
  if (normalizedMethod === 'GET' && drawdownMatch) {
    return `/api/v1/risk/portfolio/${drawdownMatch[1]}/drawdown-trend`;
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
  return withTraceHeaders({ ...headers, ...tenantHeaders(req) }, req);
}

async function forwardToRiskService(
  req: Request,
  res: Response,
  baseUrl: URL,
  riskPath: string
): Promise<void> {
  const upstreamUrl = new URL(riskPath, baseUrl);
  upstreamUrl.search = new URL(req.url, 'http://gateway.local').search;
  const upstream = await fetchUpstream('risk-engine-service', upstreamUrl.toString(), {
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

function shouldForwardBody(method: string): boolean {
  return !['GET', 'HEAD'].includes(method.toUpperCase());
}
