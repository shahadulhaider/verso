import { describe, it, expect } from "vitest";
import { createEnvelope, marshalEnvelope, unmarshalEnvelope } from "./index.js";

describe("createEnvelope", () => {
  it("generates a valid ULID eventId", () => {
    const payload = new TextEncoder().encode('{"foo":"bar"}');
    const env = createEnvelope("book.created", "catalog-svc", "book-123", payload);

    // ULID is 26 uppercase alphanumeric chars
    expect(env.eventId).toMatch(/^[0-9A-Z]{26}$/);
  });

  it("sets ISO timestamp and schemaVersion 1", () => {
    const payload = new TextEncoder().encode("{}");
    const env = createEnvelope("book.created", "catalog-svc", "book-123", payload);

    expect(new Date(env.occurredAt).toISOString()).toBe(env.occurredAt);
    expect(env.schemaVersion).toBe(1);
  });

  it("preserves all fields", () => {
    const payload = new TextEncoder().encode("hello");
    const env = createEnvelope("user.registered", "auth-svc", "user-42", payload, "trace-abc");

    expect(env.type).toBe("user.registered");
    expect(env.producer).toBe("auth-svc");
    expect(env.partitionKey).toBe("user-42");
    expect(env.traceId).toBe("trace-abc");
    expect(env.payload).toEqual(payload);
  });
});

describe("marshal / unmarshal roundtrip", () => {
  it("roundtrips an envelope through JSON", () => {
    const payload = new TextEncoder().encode('{"title":"Dune"}');
    const original = createEnvelope("book.created", "catalog-svc", "book-1", payload);

    const json = marshalEnvelope(original);
    const restored = unmarshalEnvelope(json);

    expect(restored.eventId).toBe(original.eventId);
    expect(restored.type).toBe(original.type);
    expect(restored.occurredAt).toBe(original.occurredAt);
    expect(restored.producer).toBe(original.producer);
    expect(restored.partitionKey).toBe(original.partitionKey);
    expect(restored.schemaVersion).toBe(original.schemaVersion);
    expect(restored.payload).toEqual(original.payload);
  });

  it("marshalEnvelope produces valid JSON with base64 payload", () => {
    const payload = new TextEncoder().encode("binary data");
    const env = createEnvelope("test.event", "test-svc", "key-1", payload);
    const json = marshalEnvelope(env);
    const parsed = JSON.parse(json);

    expect(typeof parsed.payload).toBe("string");
    // Verify it's valid base64
    expect(Buffer.from(parsed.payload, "base64").toString()).toBe("binary data");
  });
});
