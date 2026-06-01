import cors from 'cors';
import express from 'express';
import helmet from 'helmet';
import pino from 'pino';
import pinoHttp from 'pino-http';
import { correlationIdMiddleware, CORRELATION_ID_HEADER } from './middleware/correlation-id';
import { errorHandler, notFoundHandler } from './middleware/error-handler';
import { metricsHandler, metricsMiddleware } from './observability/metrics';
import { authProxyRouter } from './routes/auth-proxy';
import { healthRouter } from './routes/health';
import { marketProxyRouter } from './routes/market-proxy';
import { orderProxyRouter } from './routes/order-proxy';
import { portfolioProxyRouter } from './routes/portfolio-proxy';

const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  base: {
    service: 'api-gateway',
    version: process.env.npm_package_version || '0.1.0'
  }
});

export function createApp() {
  const app = express();

  app.disable('x-powered-by');
  app.use(helmet());
  app.use(cors());
  app.use(express.json({ limit: '1mb' }));
  app.use(correlationIdMiddleware);
  app.use(pinoHttp({
    logger,
    customProps: (req) => ({
      correlationId: req.headers[CORRELATION_ID_HEADER]
    })
  }));
  app.use(metricsMiddleware);

  app.use(healthRouter);
  app.get('/metrics', metricsHandler);
  app.use('/api/auth', authProxyRouter());
  app.use('/api/market', marketProxyRouter());
  app.use('/api/orders', orderProxyRouter());
  app.use('/api/portfolio', portfolioProxyRouter());

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

if (require.main === module) {
  const port = Number(process.env.PORT || 8080);
  const app = createApp();

  app.listen(port, '0.0.0.0', () => {
    logger.info({ port }, 'API Gateway started');
  });
}
