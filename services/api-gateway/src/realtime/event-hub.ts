import { WebSocket } from 'ws';
import { v4 as uuidv4 } from 'uuid';
import {
  recordWebSocketKafkaEventConsumed,
  recordWebSocketMessageFailed,
  recordWebSocketMessageSent
} from '../observability/metrics';
import { streamsForTopic, WebSocketStream } from './streams';

export interface StreamMessage {
  type: string;
  topic: string;
  correlationId: string;
  timestamp: string;
  payload: unknown;
}

interface Subscriber {
  socket: WebSocket;
  stream: WebSocketStream;
  tenantId: string;
  roles: string[];
}

export class WebSocketEventHub {
  private readonly subscribers = new Set<Subscriber>();

  subscribe(socket: WebSocket, stream: WebSocketStream, tenantId = 'default-tenant', roles: string[] = []): () => void {
    const subscriber = { socket, stream, tenantId, roles };
    this.subscribers.add(subscriber);
    return () => {
      this.subscribers.delete(subscriber);
    };
  }

  subscriberCount(): number {
    return this.subscribers.size;
  }

  handleKafkaMessage(topic: string, value: Buffer | string | null | undefined): StreamMessage | undefined {
    try {
      const payloadText = Buffer.isBuffer(value) ? value.toString('utf8') : value || '{}';
      const payload = payloadText.trim().length > 0 ? JSON.parse(payloadText) : {};
      const correlationId = extractCorrelationId(payload) || uuidv4();
      const message: StreamMessage = {
        type: topic,
        topic,
        correlationId,
        timestamp: new Date().toISOString(),
        payload
      };

      recordWebSocketKafkaEventConsumed(topic);
      this.broadcast(message);
      return message;
    } catch {
      return undefined;
    }
  }

  broadcast(message: StreamMessage): void {
    const targetStreams = streamsForTopic(message.topic);
    for (const subscriber of this.subscribers) {
      if (!targetStreams.includes(subscriber.stream)) {
        continue;
      }
      if (!canReceiveTenantEvent(subscriber, message.payload)) {
        continue;
      }

      try {
        subscriber.socket.send(JSON.stringify(message));
        recordWebSocketMessageSent(subscriber.stream, message.topic);
      } catch {
        recordWebSocketMessageFailed(subscriber.stream, message.topic);
      }
    }
  }
}

function canReceiveTenantEvent(subscriber: Subscriber, payload: unknown): boolean {
  const eventTenantId = extractTenantId(payload);
  if (!eventTenantId) {
    return true;
  }
  return subscriber.tenantId === eventTenantId || subscriber.roles.includes('trading_admin');
}

function extractTenantId(payload: unknown): string | undefined {
  if (!payload || typeof payload !== 'object') {
    return undefined;
  }

  const value = (payload as Record<string, unknown>).tenantId;
  return typeof value === 'string' && value.trim().length > 0 ? value : undefined;
}

function extractCorrelationId(payload: unknown): string | undefined {
  if (!payload || typeof payload !== 'object') {
    return undefined;
  }

  const record = payload as Record<string, unknown>;
  const value = record.correlationId || record.correlation_id;
  return typeof value === 'string' && value.trim().length > 0 ? value : undefined;
}
