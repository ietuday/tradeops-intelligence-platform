import { NextFunction, Request, Response } from 'express';
import { parsePrincipal, resolveTenantId } from './tenant';
import { AdminContext } from '../types/admin';

const READ_ROLES = new Set(['trading_admin', 'risk_manager']);
const MUTATE_ROLES = new Set(['trading_admin']);

declare global {
  namespace Express {
    interface Request {
      adminContext?: AdminContext;
    }
  }
}

export function requireAdminRead(req: Request, res: Response, next: NextFunction): void {
  requireAdminRole(req, res, next, READ_ROLES);
}

export function requireAdminMutation(req: Request, res: Response, next: NextFunction): void {
  requireAdminRole(req, res, next, MUTATE_ROLES);
}

function requireAdminRole(req: Request, res: Response, next: NextFunction, allowedRoles: Set<string>): void {
  const authorization = req.header('authorization') || '';
  const principal = parsePrincipal(authorization);

  if (!authorization || !principal) {
    res.status(401).json({
      error: {
        code: 'ADMIN_AUTH_REQUIRED',
        message: 'Authentication is required for admin operations.'
      }
    });
    return;
  }

  if (!principal.roles.some((role) => allowedRoles.has(role))) {
    res.status(403).json({
      error: {
        code: 'ADMIN_FORBIDDEN',
        message: 'The caller is not allowed to access admin operations.'
      }
    });
    return;
  }

  const requestedTenant = typeof req.query.tenantId === 'string' ? req.query.tenantId : req.header('x-tenant-id');
  req.adminContext = {
    tenantId: resolveTenantId(principal, requestedTenant),
    userId: principal.userId,
    roles: principal.roles,
    authorization,
    correlationId: req.header('x-correlation-id')
  };
  next();
}
