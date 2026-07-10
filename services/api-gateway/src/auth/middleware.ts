import { NextFunction, Request, Response } from 'express';
import { CORRELATION_ID_HEADER } from '../middleware/correlation-id';
import { recordAuthRequest, recordForbidden } from '../observability/metrics';
import { IdentityContext, TRADEOPS_IDENTITY_HEADERS } from './identity-context';
import { JwksJwtValidator, JwtClaims } from './jwks';
import { isAuthorized, policyFor } from './route-policy';

let validator: JwksJwtValidator | undefined;

export function authMiddleware(req: Request, res: Response, next: NextFunction): void {
  for (const header of TRADEOPS_IDENTITY_HEADERS) {
    delete req.headers[header];
  }

  const policy = policyFor(req);
  if (policy.public || authDisabled()) {
    next();
    return;
  }

  void authenticate(req)
    .then((identity) => {
      if (!identity) {
        recordAuthRequest('jwt', 'deny', 'missing_token');
        unauthorized(req, res);
        return;
      }
      req.identity = identity;

      if (requireTenant() && identity.authType === 'jwt' && !identity.tenantId) {
        recordAuthRequest(identity.authType, 'deny', 'missing_tenant');
        recordForbidden(policy.routeGroup, 'missing_tenant');
        forbidden(req, res);
        return;
      }

      const decision = isAuthorized(identity, policy);
      if (!decision.ok) {
        recordAuthRequest(identity.authType, 'deny', decision.reason);
        recordForbidden(policy.routeGroup, decision.reason);
        forbidden(req, res);
        return;
      }

      recordAuthRequest(identity.authType, 'allow', decision.reason);
      next();
    })
    .catch(() => {
      recordAuthRequest('jwt', 'deny', 'invalid_token');
      unauthorized(req, res);
    });
}

async function authenticate(req: Request): Promise<IdentityContext | undefined> {
  const serviceName = req.header('x-tradeops-service-name');
  const serviceToken = req.header('x-tradeops-service-token');
  const expectedServiceToken = process.env.SERVICE_AUTH_SHARED_SECRET || process.env.INTERNAL_SERVICE_TOKEN;
  if (expectedServiceToken && serviceToken === expectedServiceToken && serviceName) {
    return {
      subject: serviceName,
      issuer: 'tradeops-service-auth',
      audience: ['tradeops-internal'],
      roles: ['internal_admin'],
      scopes: ['internal:read', 'metrics:read'],
      authType: 'service',
      serviceName,
      correlationId: req.header(CORRELATION_ID_HEADER)
    };
  }

  const match = /^Bearer\s+(.+)$/i.exec(req.header('authorization') || '');
  if (!match) {
    return undefined;
  }
  const claims = await getValidator().validate(match[1]);
  return normalizeClaims(claims, req.header(CORRELATION_ID_HEADER));
}

function normalizeClaims(claims: JwtClaims, correlationId?: string): IdentityContext {
  const subject = claims.sub || claims.user_id || claims.userId || '';
  return {
    userId: claims.user_id || claims.userId || subject,
    tenantId: claims.tenant_id || claims.tenantId,
    accountId: claims.account_id || claims.accountId,
    roles: listClaim(claims.roles),
    scopes: [...listClaim(claims.scopes), ...listClaim(claims.scope)],
    subject,
    issuer: claims.iss || '',
    audience: Array.isArray(claims.aud) ? claims.aud : claims.aud ? [claims.aud] : [],
    tokenId: claims.jti,
    authType: 'jwt',
    correlationId
  };
}

function listClaim(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === 'string');
  }
  if (typeof value === 'string') {
    return value.split(/[,\s]+/).map((item) => item.trim()).filter(Boolean);
  }
  return [];
}

function getValidator(): JwksJwtValidator {
  if (!validator) {
    validator = new JwksJwtValidator(process.env.OIDC_JWKS_URL || 'http://identity-service:8080/.well-known/jwks.json');
  }
  return validator;
}

function authDisabled(): boolean {
  if (process.env.NODE_ENV === 'test' && process.env.AUTH_ENABLED === undefined) {
    return true;
  }
  return (process.env.AUTH_ENABLED || 'true').toLowerCase() === 'false';
}

function requireTenant(): boolean {
  return (process.env.AUTH_REQUIRE_TENANT || 'true').toLowerCase() !== 'false';
}

function unauthorized(req: Request, res: Response): void {
  res.status(401).json({ error: 'unauthorized', message: 'Missing or invalid access token', correlationId: req.header(CORRELATION_ID_HEADER) });
}

function forbidden(req: Request, res: Response): void {
  res.status(403).json({ error: 'forbidden', message: 'Insufficient permissions', correlationId: req.header(CORRELATION_ID_HEADER) });
}
