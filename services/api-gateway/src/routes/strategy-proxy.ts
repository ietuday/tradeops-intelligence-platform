import { Router, Request, Response, NextFunction } from 'express';
import { fetchUpstream } from './proxy-utils';

const DEFAULT_STRATEGY_SERVICE_URL = 'http://strategy-service:8080';
const UUID_PATTERN = '[0-9a-fA-F-]{36}';

export function strategyProxyRouter(
  strategyServiceUrl = process.env.STRATEGY_SERVICE_URL || DEFAULT_STRATEGY_SERVICE_URL
) {
  const router = Router();
  const baseUrl = normalizeBaseUrl(strategyServiceUrl);

  router.use(async (req: Request, res: Response, next: NextFunction) => {
    try {
      const strategyPath = resolveStrategyPath(req.method, req.path);
      if (!strategyPath) {
        res.status(404).json({
          error: {
            code: 'STRATEGY_ROUTE_NOT_FOUND',
            message: `Strategy route ${req.method} ${req.path} is not supported.`
          }
        });
        return;
      }

      await forwardToStrategyService(req, res, baseUrl, strategyPath);
    } catch (error) {
      next(error);
    }
  });

  return router;
}

function normalizeBaseUrl(strategyServiceUrl: string): URL {
  const parsedUrl = new URL(strategyServiceUrl);
  if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
    throw new Error('STRATEGY_SERVICE_URL must use http or https');
  }
  return parsedUrl;
}

function resolveStrategyPath(method: string, requestPath: string): string | undefined {
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
    if (normalizedMethod === 'GET' || normalizedMethod === 'POST') {
      return '/strategies';
    }
  }

  const strategyMatch = new RegExp(`^/(${UUID_PATTERN})$`).exec(requestPath);
  if (normalizedMethod === 'GET' && strategyMatch) {
    return `/strategies/${strategyMatch[1]}`;
  }

  const backtestMatch = new RegExp(`^/(${UUID_PATTERN})/backtest$`).exec(requestPath);
  if (normalizedMethod === 'POST' && backtestMatch) {
    return `/strategies/${backtestMatch[1]}/backtest`;
  }

  const performanceMatch = new RegExp(`^/(${UUID_PATTERN})/performance$`).exec(requestPath);
  if (normalizedMethod === 'GET' && performanceMatch) {
    return `/strategies/${performanceMatch[1]}/performance`;
  }

  const signalsMatch = new RegExp(`^/(${UUID_PATTERN})/signals$`).exec(requestPath);
  if (normalizedMethod === 'GET' && signalsMatch) {
    return `/strategies/${signalsMatch[1]}/signals`;
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
  return headers;
}

function shouldForwardBody(method: string): boolean {
  return !['GET', 'HEAD'].includes(method.toUpperCase());
}

async function forwardToStrategyService(
  req: Request,
  res: Response,
  baseUrl: URL,
  strategyPath: string
): Promise<void> {
  const upstream = await fetchUpstream('strategy-service', new URL(strategyPath, baseUrl).toString(), {
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
          message: 'Strategy service returned an invalid JSON response.'
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
      message: 'Strategy service returned an unsupported response type.'
    }
  });
}
