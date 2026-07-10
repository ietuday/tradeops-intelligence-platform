import { Request } from 'express';
import { stripSpoofedIdentityHeaders, trustedIdentityHeaders } from '../src/auth/identity-context';
import { isAuthorized, policyFor } from '../src/auth/route-policy';

describe('auth route policy and trusted headers', () => {
  it('requires write scope for order creation', () => {
    const policy = policyFor({ method: 'POST', path: '/api/orders' } as Request);

    expect(policy.scopes).toContain('orders:write');
    expect(isAuthorized({
      subject: 'user-1',
      userId: 'user-1',
      tenantId: 'tenant-a',
      issuer: 'issuer',
      audience: ['tradeops-api'],
      roles: ['trader'],
      scopes: ['orders:read'],
      authType: 'jwt'
    }, policy)).toEqual({ ok: false, reason: 'missing_permission' });
  });

  it('allows admin audit access by role', () => {
    const policy = policyFor({ method: 'GET', path: '/api/audit/summary' } as Request);

    expect(isAuthorized({
      subject: 'admin-1',
      userId: 'admin-1',
      tenantId: 'tenant-a',
      issuer: 'issuer',
      audience: ['tradeops-api'],
      roles: ['admin'],
      scopes: [],
      authType: 'jwt'
    }, policy)).toEqual({ ok: true, reason: 'role_match' });
  });

  it('strips spoofed identity headers and rebuilds trusted ones', () => {
    const clean = stripSpoofedIdentityHeaders({
      authorization: 'Bearer token',
      'x-tradeops-user-id': 'attacker',
      'x-tradeops-service-token': 'fake'
    });

    expect(clean['x-tradeops-user-id']).toBeUndefined();
    expect(clean['x-tradeops-service-token']).toBeUndefined();

    const headers = trustedIdentityHeaders({
      header: (name: string) => name === 'x-correlation-id' ? 'corr-1' : undefined,
      identity: {
        subject: 'user-1',
        userId: 'user-1',
        tenantId: 'tenant-a',
        issuer: 'issuer',
        audience: ['tradeops-api'],
        roles: ['trader'],
        scopes: ['orders:read'],
        authType: 'jwt'
      }
    } as unknown as Request);

    expect(headers['x-tradeops-user-id']).toBe('user-1');
    expect(headers['x-tradeops-tenant-id']).toBe('tenant-a');
    expect(headers['x-tradeops-service-name']).toBe('api-gateway');
  });
});
