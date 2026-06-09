import { Kafka } from 'kafkajs';
import pino from 'pino';
import { WebSocketEventHub } from './event-hub';
import { TOPIC_STREAMS } from './streams';

export interface WebSocketKafkaConfig {
  brokers: string[];
  groupId: string;
  enabled: boolean;
}

export async function startWebSocketKafkaConsumer(
  hub: WebSocketEventHub,
  logger: pino.Logger,
  config = loadWebSocketKafkaConfig()
): Promise<void> {
  if (!config.enabled) {
    logger.info('WebSocket Kafka consumer disabled');
    return;
  }

  const kafka = new Kafka({
    clientId: 'api-gateway-websocket',
    brokers: config.brokers
  });
  const consumer = kafka.consumer({ groupId: config.groupId });
  const topics = Object.keys(TOPIC_STREAMS);

  try {
    await consumer.connect();
    for (const topic of topics) {
      await consumer.subscribe({ topic, fromBeginning: false });
    }
    await consumer.run({
      eachMessage: async ({ topic, message }) => {
        const streamed = hub.handleKafkaMessage(topic, message.value);
        if (!streamed) {
          logger.warn({ topic }, 'Failed to parse WebSocket Kafka payload');
        }
      }
    });
    logger.info({ topics, brokers: config.brokers }, 'WebSocket Kafka consumer started');
  } catch (error) {
    logger.error({ err: error }, 'WebSocket Kafka consumer failed to start');
  }
}

export function loadWebSocketKafkaConfig(): WebSocketKafkaConfig {
  return {
    enabled: (process.env.WS_ENABLED || 'true').toLowerCase() === 'true',
    brokers: (process.env.WS_KAFKA_BROKERS || 'redpanda:29092')
      .split(',')
      .map((broker) => broker.trim())
      .filter((broker) => broker.length > 0),
    groupId: process.env.WS_KAFKA_GROUP_ID || 'api-gateway-websocket'
  };
}
