import { context, trace } from '@opentelemetry/api';
import { NextFunction, Request, Response, Router } from 'express';
import { requireAdminMutation, requireAdminRead } from '../middleware/adminAuth';
import { recordAdminRequest } from '../observability/metrics';
import { AdminService } from '../services/adminService';

export function adminRouter(service = new AdminService()) {
  const router = Router();

  if ((process.env.ADMIN_API_ENABLED || 'true').toLowerCase() !== 'true') {
    router.use((_req, res) => {
      res.status(404).json({
        error: {
          code: 'ADMIN_API_DISABLED',
          message: 'Admin API is disabled.'
        }
      });
    });
    return router;
  }

  router.use(requireAdminRead);
  router.use(adminObservability);

  router.get('/health-summary', asyncHandler(async (_req, res) => {
    res.json(await service.healthSummary());
  }));

  router.get('/services', (_req, res) => {
    res.json(service.services());
  });

  router.get('/topics', (_req, res) => {
    res.json(service.topics());
  });

  router.get('/dlq-summary', (_req, res) => {
    res.json(service.dlqSummary());
  });

  router.get('/audit-summary', asyncHandler(async (req, res) => {
    res.json(await service.auditSummary(req.adminContext!, queryParams(req)));
  }));

  router.get('/alerts-summary', asyncHandler(async (req, res) => {
    res.json(await service.alertsSummary(req.adminContext!, queryParams(req)));
  }));

  router.get('/notifications-summary', asyncHandler(async (req, res) => {
    res.json(await service.notificationsSummary(req.adminContext!, queryParams(req)));
  }));

  router.get('/rule-config-summary', asyncHandler(async (req, res) => {
    res.json(await service.ruleConfigSummary(req.adminContext!, queryParams(req)));
  }));

  router.get('/platform-config', (_req, res) => {
    res.json(service.platformConfig());
  });

  router.get('/ops-checklist', (_req, res) => {
    res.json(service.opsChecklist());
  });

  router.post('/cache/refresh', requireAdminMutation, (_req, res) => {
    res.status(202).json(service.cacheRefresh());
  });

  return router;
}

function adminObservability(req: Request, res: Response, next: NextFunction): void {
  const endpoint = req.path;
  const span = trace.getSpan(context.active());
  if (span) {
    span.setAttribute('admin.endpoint', endpoint);
    span.setAttribute('tenant.id', req.adminContext?.tenantId || '');
    span.setAttribute('correlation.id', req.header('x-correlation-id') || '');
  }

  req.log?.info({
    endpoint,
    tenantId: req.adminContext?.tenantId,
    userId: req.adminContext?.userId,
    correlationId: req.header('x-correlation-id')
  }, 'Admin API request');

  res.on('finish', () => recordAdminRequest(endpoint, res.statusCode));
  next();
}

function asyncHandler(handler: (req: Request, res: Response) => Promise<void>) {
  return (req: Request, res: Response, next: NextFunction) => {
    handler(req, res).catch(next);
  };
}

function queryParams(req: Request): URLSearchParams {
  return new URL(req.originalUrl, 'http://gateway.local').searchParams;
}
