import { Kafka, type Consumer, type EachMessagePayload } from "kafkajs";

export interface EventEnvelope {
  eventId: string;
  type: string;
  occurredAt: string;
  producer: string;
  traceId?: string;
  partitionKey: string;
  payload: Record<string, unknown>;
  schemaVersion: number;
}

export type MessageHandler = (envelope: EventEnvelope) => Promise<void>;

export function createKafka(brokers: string): Kafka {
  return new Kafka({
    clientId: "verso-feed-service",
    brokers: brokers.split(","),
    retry: { retries: 5 },
  });
}

export async function createConsumer(
  kafka: Kafka,
  topics: string[],
  handlers: Map<string, MessageHandler>,
  logger: { info: (...args: unknown[]) => void; error: (...args: unknown[]) => void },
): Promise<Consumer> {
  const consumer = kafka.consumer({ groupId: "verso-feed-service" });
  await consumer.connect();

  for (const topic of topics) {
    await consumer.subscribe({ topic, fromBeginning: false });
  }

  await consumer.run({
    eachMessage: async ({ topic, message }: EachMessagePayload) => {
      if (!message.value) return;

      try {
        const envelope = JSON.parse(message.value.toString()) as EventEnvelope;
        const handler = handlers.get(topic);
        if (handler) {
          await handler(envelope);
        } else {
          logger.info({ topic }, "No handler for topic");
        }
      } catch (err) {
        logger.error({ err, topic }, "Failed to process message");
      }
    },
  });

  return consumer;
}
