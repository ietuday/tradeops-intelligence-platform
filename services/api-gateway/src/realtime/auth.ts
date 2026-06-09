import crypto from 'crypto';
import { IncomingMessage } from 'http';

export interface WebSocketPrincipal {
  userId?: string;
  tenantId: string;
  email?: string;
  roles: string[];
}

interface JwtPayload {
  sub?: string;
  tenantId?: string;
  email?: string;
  roles?: string[];
  iss?: string;
  exp?: number;
  nbf?: number;
}

export function extractToken(req: IncomingMessage, url: URL): string | undefined {
  const queryToken = url.searchParams.get('token');
  if (queryToken) {
    return queryToken;
  }

  const authorization = req.headers.authorization;
  if (!authorization) {
    return undefined;
  }

  const match = /^Bearer\s+(.+)$/i.exec(authorization);
  return match?.[1];
}

export function verifyJwt(token: string, secret = process.env.IDENTITY_JWT_SECRET || process.env.JWT_SECRET || 'local_dev_jwt_secret_change_me_123456789'): WebSocketPrincipal {
  const [encodedHeader, encodedPayload, signature] = token.split('.');
  if (!encodedHeader || !encodedPayload || !signature) {
    throw new Error('invalid token format');
  }

  const header = JSON.parse(base64UrlDecode(encodedHeader).toString('utf8')) as { alg?: string };
  if (header.alg !== 'HS256') {
    throw new Error('unsupported token algorithm');
  }

  const expected = crypto
    .createHmac('sha256', secret)
    .update(`${encodedHeader}.${encodedPayload}`)
    .digest('base64url');

  if (!crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected))) {
    throw new Error('invalid token signature');
  }

  const payload = JSON.parse(base64UrlDecode(encodedPayload).toString('utf8')) as JwtPayload;
  const now = Math.floor(Date.now() / 1000);

  if (payload.iss && payload.iss !== 'tradeops-identity-service') {
    throw new Error('invalid token issuer');
  }
  if (payload.exp && payload.exp < now) {
    throw new Error('token expired');
  }
  if (payload.nbf && payload.nbf > now) {
    throw new Error('token not active');
  }
  if (!Array.isArray(payload.roles)) {
    throw new Error('token roles missing');
  }

  return {
    userId: payload.sub,
    tenantId: payload.tenantId || 'default-tenant',
    email: payload.email,
    roles: payload.roles
  };
}

function base64UrlDecode(value: string): Buffer {
  return Buffer.from(value, 'base64url');
}
