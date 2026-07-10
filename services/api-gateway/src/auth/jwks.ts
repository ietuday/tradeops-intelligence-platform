import { createPublicKey, JsonWebKey, KeyObject, verify } from 'crypto';
import { recordAuthTokenValidation, recordJwksRefresh, setJwksCacheAge } from '../observability/metrics';

interface Jwk {
  kid?: string;
  kty: string;
  alg?: string;
  use?: string;
  n?: string;
  e?: string;
}

interface JwksResponse {
  keys?: Jwk[];
}

export interface JwtClaims {
  sub?: string;
  user_id?: string;
  userId?: string;
  tenant_id?: string;
  tenantId?: string;
  account_id?: string;
  accountId?: string;
  roles?: string[] | string;
  scopes?: string[] | string;
  scope?: string;
  iss?: string;
  aud?: string | string[];
  exp?: number;
  nbf?: number;
  iat?: number;
  jti?: string;
}

export class JwksJwtValidator {
  private keys = new Map<string, KeyObject>();
  private fetchedAt = 0;
  private readonly ttlMs: number;
  private readonly issuer: string;
  private readonly audience: string;
  private readonly algorithms: Set<string>;
  private readonly clockSkewSeconds: number;

  constructor(private readonly jwksUrl: string) {
    this.ttlMs = parseDurationMs(process.env.OIDC_JWKS_CACHE_TTL || '10m');
    this.issuer = process.env.OIDC_ISSUER_URL || 'http://identity-service:8080';
    this.audience = process.env.OIDC_AUDIENCE || 'tradeops-api';
    this.algorithms = new Set((process.env.OIDC_ALLOWED_ALGORITHMS || 'RS256').split(',').map((value) => value.trim()).filter(Boolean));
    this.clockSkewSeconds = Number(process.env.AUTH_CLOCK_SKEW_SECONDS || process.env.AUTH_CLOCK_SKEW || 60);
  }

  async validate(token: string): Promise<JwtClaims> {
    try {
      const [encodedHeader, encodedPayload, encodedSignature] = token.split('.');
      if (!encodedHeader || !encodedPayload || !encodedSignature) {
        throw new Error('malformed');
      }

      const header = JSON.parse(Buffer.from(encodedHeader, 'base64url').toString('utf8')) as { alg?: string; kid?: string };
      if (!header.alg || !this.algorithms.has(header.alg)) {
        throw new Error('unsupported_algorithm');
      }
      if (!header.kid) {
        throw new Error('missing_kid');
      }

      let key = await this.getKey(header.kid);
      if (!key) {
        await this.refreshKeys(true);
        key = await this.getKey(header.kid);
      }
      if (!key) {
        throw new Error('unknown_kid');
      }

      const validSignature = verify(signatureAlgorithm(header.alg), Buffer.from(`${encodedHeader}.${encodedPayload}`), key, Buffer.from(encodedSignature, 'base64url'));
      if (!validSignature) {
        throw new Error('invalid_signature');
      }

      const claims = JSON.parse(Buffer.from(encodedPayload, 'base64url').toString('utf8')) as JwtClaims;
      this.validateClaims(claims);
      recordAuthTokenValidation('success', 'valid');
      return claims;
    } catch (error) {
      recordAuthTokenValidation('failure', boundedReason(error));
      throw error;
    }
  }

  private validateClaims(claims: JwtClaims): void {
    const now = Math.floor(Date.now() / 1000);
    if (claims.iss !== this.issuer) {
      throw new Error('wrong_issuer');
    }
    const audiences = Array.isArray(claims.aud) ? claims.aud : claims.aud ? [claims.aud] : [];
    if (!audiences.includes(this.audience)) {
      throw new Error('wrong_audience');
    }
    if (!claims.exp || claims.exp + this.clockSkewSeconds < now) {
      throw new Error('expired');
    }
    if (claims.nbf && claims.nbf - this.clockSkewSeconds > now) {
      throw new Error('not_before');
    }
  }

  private async getKey(kid: string): Promise<KeyObject | undefined> {
    if (!this.keys.has(kid) || Date.now() - this.fetchedAt > this.ttlMs) {
      await this.refreshKeys(false);
    }
    setJwksCacheAge((Date.now() - this.fetchedAt) / 1000);
    return this.keys.get(kid);
  }

  private async refreshKeys(rotationRefresh: boolean): Promise<void> {
    try {
      const response = await fetch(this.jwksUrl);
      if (!response.ok) {
        throw new Error(`jwks_http_${response.status}`);
      }
      const jwks = await response.json() as JwksResponse;
      const nextKeys = new Map<string, KeyObject>();
      for (const key of jwks.keys || []) {
        if (key.kty === 'RSA' && key.kid) {
          nextKeys.set(key.kid, createPublicKey({ key: key as JsonWebKey, format: 'jwk' }));
        }
      }
      this.keys = nextKeys;
      this.fetchedAt = Date.now();
      recordJwksRefresh('success');
      if (rotationRefresh) {
        recordJwksRefresh('rotation');
      }
    } catch (error) {
      recordJwksRefresh('failure');
      throw error;
    }
  }
}

function signatureAlgorithm(alg: string): string {
  if (alg === 'RS256') return 'RSA-SHA256';
  throw new Error('unsupported_algorithm');
}

function boundedReason(error: unknown): string {
  const message = error instanceof Error ? error.message : 'invalid';
  if (message.includes('expired')) return 'expired';
  if (message.includes('issuer')) return 'issuer';
  if (message.includes('audience')) return 'audience';
  if (message.includes('algorithm')) return 'algorithm';
  if (message.includes('signature')) return 'signature';
  if (message.includes('kid')) return 'kid';
  if (message.includes('jwks')) return 'jwks';
  return 'invalid';
}

function parseDurationMs(value: string): number {
  const match = /^(\d+)(ms|s|m|h)?$/.exec(value.trim());
  if (!match) return 600_000;
  const amount = Number(match[1]);
  const unit = match[2] || 'ms';
  if (unit === 'h') return amount * 60 * 60 * 1000;
  if (unit === 'm') return amount * 60 * 1000;
  if (unit === 's') return amount * 1000;
  return amount;
}
