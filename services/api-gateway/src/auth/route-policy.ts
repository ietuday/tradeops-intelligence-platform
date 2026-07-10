import { Request } from 'express';
import { IdentityContext } from './identity-context';

export interface RoutePolicy {
  public?: boolean;
  scopes?: string[];
  roles?: string[];
  admin?: boolean;
  routeGroup: string;
}

const POLICIES: Array<{ method: string; pattern: RegExp; policy: RoutePolicy }> = [
  { method: 'GET', pattern: /^\/health$/, policy: { public: true, routeGroup: 'health' } },
  { method: 'GET', pattern: /^\/ready$/, policy: { public: true, routeGroup: 'health' } },
  { method: 'GET', pattern: /^\/$/, policy: { public: true, routeGroup: 'root' } },
  { method: 'GET', pattern: /^\/metrics$/, policy: { roles: ['internal_admin', 'admin'], scopes: ['metrics:read'], routeGroup: 'internal' } },
  { method: '*', pattern: /^\/api\/auth\/(login|register|refresh|token)$/, policy: { public: true, routeGroup: 'auth' } },
  { method: 'GET', pattern: /^\/api\/orders(\/.*)?$/, policy: { scopes: ['orders:read'], routeGroup: 'orders' } },
  { method: 'POST', pattern: /^\/api\/orders$/, policy: { scopes: ['orders:write'], routeGroup: 'orders' } },
  { method: 'PATCH', pattern: /^\/api\/orders\/[^/]+$/, policy: { scopes: ['orders:write'], routeGroup: 'orders' } },
  { method: 'POST', pattern: /^\/api\/orders\/[^/]+\/cancel$/, policy: { scopes: ['orders:cancel', 'orders:write'], routeGroup: 'orders' } },
  { method: 'GET', pattern: /^\/api\/portfolio(\/.*)?$/, policy: { scopes: ['portfolio:read'], routeGroup: 'portfolio' } },
  { method: 'GET', pattern: /^\/api\/audit(\/.*)?$/, policy: { roles: ['admin', 'trading_admin'], scopes: ['audit:read'], admin: true, routeGroup: 'audit' } },
  { method: '*', pattern: /^\/api\/admin(\/.*)?$/, policy: { roles: ['admin', 'trading_admin', 'internal_admin'], scopes: ['admin:read'], admin: true, routeGroup: 'admin' } },
  { method: '*', pattern: /^\/internal(\/.*)?$/, policy: { roles: ['internal_admin'], scopes: ['internal:read'], admin: true, routeGroup: 'internal' } }
];

export function policyFor(req: Request): RoutePolicy {
  const method = req.method.toUpperCase();
  for (const entry of POLICIES) {
    if ((entry.method === '*' || entry.method === method) && entry.pattern.test(req.path)) {
      return entry.policy;
    }
  }
  return { routeGroup: routeGroup(req.path) };
}

export function isAuthorized(identity: IdentityContext, policy: RoutePolicy): { ok: boolean; reason: string } {
  if (policy.public) {
    return { ok: true, reason: 'public' };
  }
  if (identity.authType === 'service') {
    return { ok: true, reason: 'service_auth' };
  }
  if (policy.roles?.some((role) => identity.roles.includes(role))) {
    return { ok: true, reason: 'role_match' };
  }
  if (policy.scopes?.some((scope) => identity.scopes.includes(scope))) {
    return { ok: true, reason: 'scope_match' };
  }
  if (!policy.roles?.length && !policy.scopes?.length) {
    return { ok: true, reason: 'authenticated' };
  }
  return { ok: false, reason: 'missing_permission' };
}

function routeGroup(path: string): string {
  if (path.startsWith('/api/orders')) return 'orders';
  if (path.startsWith('/api/portfolio')) return 'portfolio';
  if (path.startsWith('/api/audit')) return 'audit';
  if (path.startsWith('/api/admin')) return 'admin';
  if (path.startsWith('/internal') || path === '/metrics') return 'internal';
  return 'api';
}
