import { NextFunction, Request, Response } from 'express';
import { v4 as uuidv4 } from 'uuid';

export const CORRELATION_ID_HEADER = 'x-correlation-id';

export function correlationIdMiddleware(req: Request, res: Response, next: NextFunction): void {
  const incomingCorrelationId = req.header(CORRELATION_ID_HEADER);
  const correlationId = incomingCorrelationId && incomingCorrelationId.trim().length > 0
    ? incomingCorrelationId
    : uuidv4();

  res.setHeader(CORRELATION_ID_HEADER, correlationId);
  req.headers[CORRELATION_ID_HEADER] = correlationId;
  next();
}