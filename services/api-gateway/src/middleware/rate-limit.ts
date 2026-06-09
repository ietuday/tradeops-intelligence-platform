import { NextFunction, Request, Response } from 'express';
import { CORRELATION_ID_HEADER } from './correlation-id';

export interface RateLimitOptions {
  windowMs: number;
  maxRequests: number;
}

interface ClientWindow {
  count: number;
  resetAt: number;
}

export function createRateLimitMiddleware(options: RateLimitOptions) {
  const clients = new Map<string, ClientWindow>();

  return function rateLimitMiddleware(req: Request, res: Response, next: NextFunction): void {
    const now = Date.now();
    const clientKey = req.ip || req.socket.remoteAddress || 'unknown';
    const current = clients.get(clientKey);

    if (!current || current.resetAt <= now) {
      clients.set(clientKey, { count: 1, resetAt: now + options.windowMs });
      next();
      return;
    }

    current.count += 1;
    if (current.count <= options.maxRequests) {
      next();
      return;
    }

    const retryAfterSeconds = Math.max(1, Math.ceil((current.resetAt - now) / 1000));
    res.setHeader('Retry-After', String(retryAfterSeconds));
    res.status(429).json({
      error: {
        code: 'RATE_LIMIT_EXCEEDED',
        message: 'Too many requests. Please retry after the rate limit window resets.',
        correlationId: req.headers[CORRELATION_ID_HEADER]
      }
    });
  };
}
