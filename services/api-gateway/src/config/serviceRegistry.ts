import { DlqEntry, ServiceRegistryEntry, TopicCatalogEntry } from '../types/admin';

export function serviceRegistry(): ServiceRegistryEntry[] {
  return [
    service('api-gateway', 'node', 'edge', 'http://api-gateway:8080', true, false, [
      'websocket-stream-message'
    ], [
      'market.ticks',
      'order.created',
      'order.validated',
      'order.accepted',
      'order.filled',
      'order.rejected',
      'order.cancelled',
      'surveillance.alert.created',
      'surveillance.alert.acknowledged',
      'surveillance.alert.resolved',
      'surveillance.alert.dismissed',
      'notification.created',
      'notification.sent',
      'notification.failed',
      'notification.read',
      'notification.retry_requested',
      'audit.log.created'
    ]),
    service('identity-service', 'go', 'security', process.env.IDENTITY_SERVICE_URL || 'http://identity-service:8080', true, true, [], []),
    service('market-data-service', 'go', 'market-data', process.env.MARKET_DATA_SERVICE_URL || 'http://market-data-service:8080', true, true, ['market.ticks'], []),
    service('order-service', 'go', 'trading', process.env.ORDER_SERVICE_URL || 'http://order-service:8080', true, true, [
      'order.created',
      'order.validated',
      'order.accepted',
      'order.filled',
      'order.rejected',
      'order.cancelled'
    ], []),
    service('portfolio-service', 'go', 'portfolio', process.env.PORTFOLIO_SERVICE_URL || 'http://portfolio-service:8080', true, true, [
      'portfolio.updated',
      'portfolio.snapshot.created'
    ], ['order.filled']),
    service('strategy-service', 'python', 'strategy', process.env.STRATEGY_SERVICE_URL || 'http://strategy-service:8080', false, false, [
      'strategy.signal.generated',
      'strategy.backtest.completed'
    ], []),
    service('risk-engine-service', 'python', 'risk', process.env.RISK_SERVICE_URL || 'http://risk-engine-service:8080', false, false, [
      'risk.score.updated',
      'risk.breached',
      'risk.anomaly.detected',
      'risk.recommendation.created'
    ], []),
    service('surveillance-service', 'go', 'surveillance', process.env.SURVEILLANCE_SERVICE_URL || 'http://surveillance-service:8090', true, false, [
      'surveillance.alert.created',
      'surveillance.alert.acknowledged',
      'surveillance.alert.resolved',
      'surveillance.alert.dismissed',
      'surveillance.rule_config.updated',
      'surveillance.rule_config.enabled',
      'surveillance.rule_config.disabled',
      'surveillance.rule_simulation.requested',
      'surveillance.rule_simulation.completed',
      'surveillance.rule_simulation.failed'
    ], ['market.ticks', 'order.created', 'order.cancelled', 'order.filled', 'portfolio.updated', 'strategy.signal.generated', 'risk.score.updated']),
    service('notification-service', 'go', 'notifications', process.env.NOTIFICATION_SERVICE_URL || 'http://notification-service:8091', true, false, [
      'notification.created',
      'notification.sent',
      'notification.failed',
      'notification.read',
      'notification.retry_requested'
    ], ['surveillance.alert.created', 'surveillance.alert.acknowledged', 'surveillance.alert.resolved', 'surveillance.alert.dismissed']),
    service('audit-service', 'go', 'audit', process.env.AUDIT_SERVICE_URL || 'http://audit-service:8092', true, false, [
      'audit.log.created'
    ], ['order.created', 'order.filled', 'order.cancelled', 'risk.score.updated', 'risk.breached', 'surveillance.alert.created', 'notification.sent'])
  ];
}

export const topicCatalog: TopicCatalogEntry[] = [
  topic('market.ticks', 'market-data-service', ['surveillance-service', 'api-gateway'], 'schemas/events/market/market.ticks.v1.json', 'Normalized market tick event.'),
  topic('order.created', 'order-service', ['surveillance-service', 'audit-service', 'api-gateway'], 'schemas/events/orders/order.created.v1.json', 'Order submitted/created event.'),
  topic('order.validated', 'order-service', ['api-gateway'], 'schemas/events/orders/order.validated.v1.json', 'Order validation lifecycle event.'),
  topic('order.accepted', 'order-service', ['api-gateway'], 'schemas/events/orders/order.accepted.v1.json', 'Accepted order lifecycle event.'),
  topic('order.filled', 'order-service', ['portfolio-service', 'surveillance-service', 'audit-service', 'api-gateway'], 'schemas/events/orders/order.filled.v1.json', 'Filled order event.'),
  topic('order.rejected', 'order-service', ['api-gateway'], 'schemas/events/orders/order.rejected.v1.json', 'Rejected order lifecycle event.'),
  topic('order.cancelled', 'order-service', ['surveillance-service', 'audit-service', 'api-gateway'], 'schemas/events/orders/order.cancelled.v1.json', 'Cancelled order event.'),
  topic('portfolio.updated', 'portfolio-service', ['surveillance-service', 'audit-service'], 'schemas/events/portfolio/portfolio.updated.v1.json', 'Portfolio holdings/cash update event.'),
  topic('portfolio.snapshot.created', 'portfolio-service', [], 'schemas/events/portfolio/portfolio.snapshot.created.v1.json', 'Portfolio snapshot event.'),
  topic('strategy.signal.generated', 'strategy-service', ['surveillance-service'], 'schemas/events/strategy/strategy.signal.generated.v1.json', 'Strategy signal event.'),
  topic('strategy.backtest.completed', 'strategy-service', [], 'schemas/events/strategy/strategy.backtest.completed.v1.json', 'Backtest completion event.'),
  topic('risk.score.updated', 'risk-engine-service', ['surveillance-service', 'audit-service'], 'schemas/events/risk/risk.score.updated.v1.json', 'Risk score update event.'),
  topic('risk.breached', 'risk-engine-service', ['audit-service'], 'schemas/events/risk/risk.breached.v1.json', 'Risk threshold breach event.'),
  topic('surveillance.alert.created', 'surveillance-service', ['notification-service', 'audit-service', 'api-gateway'], 'schemas/events/surveillance/surveillance.alert.created.v1.json', 'New surveillance alert event.'),
  topic('surveillance.rule_config.updated', 'surveillance-service', ['future audit/compliance integrations'], 'schemas/events/surveillance/surveillance.rule_config.updated.v1.json', 'Surveillance rule config changed event.'),
  topic('notification.created', 'notification-service', ['api-gateway'], 'schemas/events/notifications/notification.created.v1.json', 'Notification created event.'),
  topic('notification.failed', 'notification-service', ['audit-service', 'api-gateway'], 'schemas/events/notifications/notification.failed.v1.json', 'Notification delivery failure event.'),
  topic('audit.log.created', 'audit-service', ['api-gateway'], 'schemas/events/audit/audit.log.created.v1.json', 'Audit log created event.')
];

export const dlqCatalog: DlqEntry[] = [
  dlq('portfolio.dlq', 'portfolio-service', 'Failed portfolio event processing.'),
  dlq('surveillance.dlq', 'surveillance-service', 'Failed surveillance event processing.'),
  dlq('notification.dlq', 'notification-service', 'Failed notification delivery event processing.'),
  dlq('audit.dlq', 'audit-service', 'Failed audit event processing.')
];

function service(
  name: string,
  type: ServiceRegistryEntry['type'],
  category: string,
  baseUrl: string,
  ownsData: boolean,
  critical: boolean,
  producesTopics: string[],
  consumesTopics: string[]
): ServiceRegistryEntry {
  return {
    name,
    type,
    category,
    baseUrl,
    healthPath: '/health',
    readyPath: '/ready',
    metricsPath: '/metrics',
    ownsData,
    critical,
    producesTopics,
    consumesTopics
  };
}

function topic(topicName: string, producer: string, consumers: string[], schema: string, description: string): TopicCatalogEntry {
  return {
    topic: topicName,
    producer,
    consumers,
    schema,
    version: '1.0',
    description
  };
}

function dlq(topicName: string, owner: string, description: string): DlqEntry {
  return {
    topic: topicName,
    owner,
    description,
    replayScript: 'scripts/replay-dlq-events.sh',
    runbook: 'docs/reliability/dead-letter-topics.md',
    status: 'STATIC_CATALOG'
  };
}
