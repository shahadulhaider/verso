import { ulid } from "ulid";

/** Canonical event envelope — matches Go EventEnvelope struct. */
export interface EventEnvelope {
  eventId: string;
  type: string;
  occurredAt: string;
  producer: string;
  traceId?: string;
  partitionKey: string;
  payload: Uint8Array;
  schemaVersion: number;
}

/**
 * Create a new EventEnvelope with a ULID id and ISO timestamp.
 * Payload is passed as Uint8Array (pre-serialized protobuf or JSON bytes).
 */
export function createEnvelope(
  type: string,
  producer: string,
  partitionKey: string,
  payload: Uint8Array,
  traceId?: string,
): EventEnvelope {
  return {
    eventId: ulid(),
    type,
    occurredAt: new Date().toISOString(),
    producer,
    traceId,
    partitionKey,
    payload,
    schemaVersion: 1,
  };
}

/**
 * Serialize an envelope to JSON string.
 * Payload bytes are base64-encoded for JSON transport.
 */
export function marshalEnvelope(envelope: EventEnvelope): string {
  return JSON.stringify({
    ...envelope,
    payload: Buffer.from(envelope.payload).toString("base64"),
  });
}

/**
 * Deserialize a JSON string back to an EventEnvelope.
 * Decodes base64 payload back to Uint8Array.
 */
export function unmarshalEnvelope(json: string): EventEnvelope {
  const parsed = JSON.parse(json) as EventEnvelope & { payload: string };
  return {
    ...parsed,
    payload: Uint8Array.from(Buffer.from(parsed.payload as string, "base64")),
  };
}
