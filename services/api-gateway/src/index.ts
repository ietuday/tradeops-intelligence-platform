import cors from 'cors';
import express from 'express';
import { createServer } from 'http';
import helmet from 'helmet';
import pino from 'pino';
import pinoHttp from 'pino-http';
import { correlationIdMiddleware, CORRELATION_ID_HEADER } from './middleware/correlation-id';
import { errorHandler, notFoundHandler } from './middleware/error-handler';
import { createRateLimitMiddleware } from './middleware/rate-limit';
import { metricsHandler, metricsMiddleware } from './observability/metrics';
import { authProxyRouter } from './routes/auth-proxy';
import { auditProxyRouter } from './routes/audit-proxy';
import { healthRouter } from './routes/health';
import { marketProxyRouter } from './routes/market-proxy';
import { notificationsProxyRouter } from './routes/notifications-proxy';
import { orderProxyRouter } from './routes/order-proxy';
import { portfolioProxyRouter } from './routes/portfolio-proxy';
import { riskProxyRouter } from './routes/risk-proxy';
import { strategyProxyRouter } from './routes/strategy-proxy';
import { surveillanceProxyRouter } from './routes/surveillance-proxy';
import { startWebSocketKafkaConsumer } from './realtime/kafka-consumer';
import { attachWebSocketServer } from './realtime/websocket-server';

const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  base: {
    service: 'api-gateway',
    version: process.env.npm_package_version || '0.1.0'
  }
});

export function createApp() {
  const app = express();
  const requestBodyLimit = process.env.REQUEST_BODY_LIMIT || '1mb';
  const rateLimitWindowMs = Number(process.env.RATE_LIMIT_WINDOW_MS || 60000);
  const rateLimitMaxRequests = Number(process.env.RATE_LIMIT_MAX_REQUESTS || 300);
  const corsOrigins = parseCorsOrigins(process.env.CORS_ORIGIN);

  app.disable('x-powered-by');
  app.use(helmet());
  app.use(cors({
    origin: corsOrigins.length > 0 ? corsOrigins : true
  }));
  app.use(correlationIdMiddleware);
  app.use(pinoHttp({
    logger,
    customProps: (req) => ({
      correlationId: req.headers[CORRELATION_ID_HEADER]
    })
  }));
  app.use(metricsMiddleware);
  app.use(createRateLimitMiddleware({
    windowMs: Number.isFinite(rateLimitWindowMs) && rateLimitWindowMs > 0 ? rateLimitWindowMs : 60000,
    maxRequests: Number.isFinite(rateLimitMaxRequests) && rateLimitMaxRequests > 0 ? rateLimitMaxRequests : 300
  }));
  app.use(express.json({ limit: requestBodyLimit }));

  app.use(healthRouter);
  app.get('/metrics', metricsHandler);
  app.use('/api/audit', auditProxyRouter());
  app.use('/api/auth', authProxyRouter());
  app.use('/api/market', marketProxyRouter());
  app.use('/api/notifications', notificationsProxyRouter());
  app.use('/api/orders', orderProxyRouter());
  app.use('/api/portfolio', portfolioProxyRouter());
  app.use('/api/strategies', strategyProxyRouter());
  app.use('/api/risk', riskProxyRouter());
  app.use('/api/surveillance', surveillanceProxyRouter());

  app.get('/', (_req, res) => {
    res.status(200).json({
      service: 'api-gateway',
      platform: 'TradeOps Intelligence Platform',
      version: '0.1.0'
    });
  });

  app.use(notFoundHandler);
  app.use(errorHandler);

  return app;
}

function parseCorsOrigins(value: string | undefined): string[] {
  if (!value) {
    return [];
  }

  return value
    .split(',')
    .map((origin) => origin.trim())
    .filter((origin) => origin.length > 0);
}

if (require.main === module) {
  const port = Number(process.env.PORT || 8080);
  const app = createApp();
  const server = createServer(app);

  if ((process.env.WS_ENABLED || 'true').toLowerCase() === 'true') {
    const websocket = attachWebSocketServer(server, logger);
    void startWebSocketKafkaConsumer(websocket.hub, logger);
  }

  server.listen(port, '0.0.0.0', () => {
    logger.info({ port }, 'API Gateway started');
  });
}
