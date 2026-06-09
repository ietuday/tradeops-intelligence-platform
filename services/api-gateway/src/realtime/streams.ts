export type WebSocketStream = 'all' | 'market' | 'orders' | 'alerts' | 'notifications' | 'audit';

export const STREAM_PATHS: Record<string, WebSocketStream> = {
  '/ws': 'all',
  '/ws/market': 'market',
  '/ws/orders': 'orders',
  '/ws/alerts': 'alerts',
  '/ws/notifications': 'notifications',
  '/ws/audit': 'audit'
};

export const TOPIC_STREAMS: Record<string, WebSocketStream[]> = {
  'market.ticks': ['market'],
  'order.created': ['orders'],
  'order.validated': ['orders'],
  'order.accepted': ['orders'],
  'order.filled': ['orders'],
  'order.rejected': ['orders'],
  'order.cancelled': ['orders'],
  'surveillance.alert.created': ['alerts'],
  'surveillance.alert.acknowledged': ['alerts'],
  'surveillance.alert.resolved': ['alerts'],
  'notification.created': ['notifications'],
  'notification.sent': ['notifications'],
  'notification.failed': ['notifications'],
  'notification.read': ['notifications'],
  'audit.log.created': ['audit']
};

const STREAM_ROLES: Record<WebSocketStream, string[]> = {
  all: ['trading_admin'],
  market: ['trading_admin', 'trader', 'risk_manager', 'analyst', 'viewer'],
  orders: ['trading_admin', 'trader'],
  alerts: ['trading_admin', 'risk_manager', 'analyst', 'viewer'],
  notifications: ['trading_admin', 'trader', 'risk_manager', 'analyst'],
  audit: ['trading_admin', 'risk_manager', 'analyst']
};

export function resolveStream(path: string): WebSocketStream | undefined {
  return STREAM_PATHS[path];
}

export function topicsForStream(stream: WebSocketStream): string[] {
  if (stream === 'all') {
    return Object.keys(TOPIC_STREAMS);
  }

  return Object.entries(TOPIC_STREAMS)
    .filter(([, streams]) => streams.includes(stream))
    .map(([topic]) => topic);
}

export function streamsForTopic(topic: string): WebSocketStream[] {
  return ['all', ...(TOPIC_STREAMS[topic] || [])];
}

export function canAccessStream(roles: string[], stream: WebSocketStream): boolean {
  const allowed = STREAM_ROLES[stream];
  return roles.some((role) => allowed.includes(role));
}
