import { Request } from 'express';

export interface TenantPrincipal {
  tenantId?: string;
  roles: string[];
}

export const DEFAULT_TENANT_ID = process.env.DEFAULT_TENANT_ID || 'default-tenant';

export function tenantHeaders(req: Request): Record<string, string> {
  const principal = parsePrincipal(req.header('authorization'));
  const externalTenant = req.header('x-tenant-id')?.trim();

  const tenantId = resolveTenantId(principal, externalTenant);
  return tenantId ? { 'x-tenant-id': tenantId } : {};
}

export function resolveTenantId(principal: TenantPrincipal | undefined, externalTenant?: string): string {
  if (principal?.tenantId) {
    if (externalTenant && principal.roles.includes('trading_admin')) {
      return externalTenant;
    }
    return principal.tenantId;
  }
  return externalTenant || DEFAULT_TENANT_ID;
}

export function parsePrincipal(authorization?: string): TenantPrincipal | undefined {
  const match = /^Bearer\s+(.+)$/i.exec(authorization || '');
  if (!match) {
    return undefined;
  }

  const parts = match[1].split('.');
  if (parts.length < 2) {
    return undefined;
  }

  try {
    const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString('utf8')) as {
      tenantId?: unknown;
      roles?: unknown;
    };
    return {
      tenantId: typeof payload.tenantId === 'string' ? payload.tenantId : undefined,
      roles: Array.isArray(payload.roles) ? payload.roles.filter((role): role is string => typeof role === 'string') : []
    };
  } catch {
    return undefined;
  }
}
