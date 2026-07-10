import { Request } from 'express';

export interface IdentityContext {
  userId?: string;
  tenantId?: string;
  accountId?: string;
  roles: string[];
  scopes: string[];
  subject: string;
  issuer: string;
  audience: string[];
  tokenId?: string;
  authType: 'jwt' | 'service';
  serviceName?: string;
  correlationId?: string;
}

declare global {
  namespace Express {
    interface Request {
      identity?: IdentityContext;
    }
  }
}

export const TRADEOPS_IDENTITY_HEADERS = [
  'x-tradeops-user-id',
  'x-tradeops-tenant-id',
  'x-tradeops-account-id',
  'x-tradeops-roles',
  'x-tradeops-scopes',
  'x-tradeops-subject',
  'x-tradeops-auth-type',
  'x-tradeops-service-name'
];

export function trustedIdentityHeaders(req: Request): Record<string, string> {
  const identity = req.identity;
  const headers: Record<string, string> = {};
  const correlationId = req.header('x-correlation-id') || identity?.correlationId;
  const serviceToken = process.env.SERVICE_AUTH_SHARED_SECRET || process.env.INTERNAL_SERVICE_TOKEN || 'local-dev-service-secret';

  if (correlationId) {
    headers['x-correlation-id'] = correlationId;
  }

  headers['x-tradeops-service-token'] = serviceToken;
  headers['x-tradeops-service-name'] = 'api-gateway';

  if (!identity) {
    return headers;
  }

  if (identity.userId) headers['x-tradeops-user-id'] = identity.userId;
  if (identity.tenantId) headers['x-tradeops-tenant-id'] = identity.tenantId;
  if (identity.accountId) headers['x-tradeops-account-id'] = identity.accountId;
  headers['x-tradeops-roles'] = identity.roles.join(' ');
  headers['x-tradeops-scopes'] = identity.scopes.join(' ');
  headers['x-tradeops-subject'] = identity.subject;
  headers['x-tradeops-auth-type'] = identity.authType;
  return headers;
}

export function stripSpoofedIdentityHeaders(headers: Record<string, string>): Record<string, string> {
  const clean = { ...headers };
  for (const header of TRADEOPS_IDENTITY_HEADERS) {
    delete clean[header];
  }
  delete clean['x-tradeops-service-token'];
  return clean;
}
