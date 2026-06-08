import { NextFunction, Request, Response } from 'express';
import { CORRELATION_ID_HEADER } from './correlation-id';
import { isProxyError } from '../routes/proxy-utils';

export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
    correlationId?: string | string[];
  };
}

export function notFoundHandler(req: Request, res: Response): void {
  res.status(404).json({
    error: {
      code: 'ROUTE_NOT_FOUND',
      message: `Route ${req.method} ${req.path} was not found.`,
      correlationId: req.headers[CORRELATION_ID_HEADER]
    }
  });
}

export function errorHandler(error: Error, req: Request, res: Response, _next: NextFunction): void {
  req.log?.error({ err: error }, 'Unhandled API Gateway error');

  if (isProxyError(error)) {
    res.status(error.status).json({
      error: {
        code: error.code,
        message: error.message,
        correlationId: req.headers[CORRELATION_ID_HEADER]
      }
    });
    return;
  }

  const response: ApiErrorResponse = {
    error: {
      code: 'INTERNAL_SERVER_ERROR',
      message: 'An unexpected error occurred.',
      correlationId: req.headers[CORRELATION_ID_HEADER]
    }
  };

  res.status(500).json(response);
}
