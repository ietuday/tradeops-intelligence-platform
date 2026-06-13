import { config } from '../config';
import { RealtimeEvent } from '../types';

export type StreamName = 'all' | 'market' | 'orders' | 'alerts' | 'notifications' | 'audit';
export type WebSocketStatus = 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR';

const streamPaths: Record<StreamName, string> = {
  all: '/ws',
  market: '/ws/market',
  orders: '/ws/orders',
  alerts: '/ws/alerts',
  notifications: '/ws/notifications',
  audit: '/ws/audit'
};

export interface DashboardSocket {
  disconnect: () => void;
}

export function connectDashboardSocket(args: {
  stream: StreamName;
  token: string;
  onStatus: (status: WebSocketStatus) => void;
  onEvent: (event: RealtimeEvent) => void;
}): DashboardSocket {
  let socket: WebSocket | undefined;
  let stopped = false;
  let attempts = 0;

  const connect = () => {
    args.onStatus('CONNECTING');
    const url = new URL(streamPaths[args.stream], config.wsBaseUrl);
    if (args.token) {
      url.searchParams.set('token', args.token);
    }
    url.searchParams.set('correlationId', `dashboard-ws-${Date.now()}`);
    socket = new WebSocket(url.toString());

    socket.onopen = () => {
      attempts = 0;
      args.onStatus('CONNECTED');
    };
    socket.onmessage = (message) => {
      try {
        args.onEvent(JSON.parse(message.data) as RealtimeEvent);
      } catch {
        args.onEvent({ type: 'unparsed', payload: message.data, timestamp: new Date().toISOString() });
      }
    };
    socket.onerror = () => args.onStatus('ERROR');
    socket.onclose = () => {
      args.onStatus('DISCONNECTED');
      if (!stopped) {
        attempts += 1;
        window.setTimeout(connect, Math.min(1000 * attempts, 5000));
      }
    };
  };

  connect();
  return {
    disconnect: () => {
      stopped = true;
      socket?.close();
    }
  };
}
