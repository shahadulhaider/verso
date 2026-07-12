import { describe, it, expect, vi } from "vitest";
import { CREATE_TABLE_SQL, insertEvent, markDelivered, pendingEvents } from "./index.js";

// Minimal pg mock types
function createMockClient() {
  return {
    query: vi.fn().mockResolvedValue({ rows: [], rowCount: 0 }),
  };
}

function createMockPool() {
  return {
    query: vi.fn().mockResolvedValue({ rows: [], rowCount: 0 }),
  };
}

describe("CREATE_TABLE_SQL", () => {
  it("contains the expected table name and columns", () => {
    expect(CREATE_TABLE_SQL).toContain("outbox_events");
    expect(CREATE_TABLE_SQL).toContain("event_id");
    expect(CREATE_TABLE_SQL).toContain("aggregate_type");
    expect(CREATE_TABLE_SQL).toContain("aggregate_id");
    expect(CREATE_TABLE_SQL).toContain("event_type");
    expect(CREATE_TABLE_SQL).toContain("payload");
    expect(CREATE_TABLE_SQL).toContain("JSONB");
    expect(CREATE_TABLE_SQL).toContain("CHAR(26)");
    expect(CREATE_TABLE_SQL).toContain("delivered");
  });
});

describe("insertEvent", () => {
  it("executes INSERT with correct parameters", async () => {
    const client = createMockClient();
    const envelope = {
      eventId: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      type: "book.created",
      occurredAt: "2024-01-01T00:00:00.000Z",
      producer: "catalog-svc",
    };

    await insertEvent(client as any, "book", "book-123", envelope);

    expect(client.query).toHaveBeenCalledOnce();
    const [sql, params] = client.query.mock.calls[0];
    expect(sql).toContain("INSERT INTO outbox_events");
    expect(params[0]).toBe("01ARZ3NDEKTSV4RRFFQ69G5FAV");
    expect(params[1]).toBe("book");
    expect(params[2]).toBe("book-123");
    expect(params[3]).toBe("book.created");
    expect(JSON.parse(params[4])).toEqual(envelope);
  });
});

describe("markDelivered", () => {
  it("executes UPDATE with event_id", async () => {
    const pool = createMockPool();
    await markDelivered(pool as any, "01ARZ3NDEKTSV4RRFFQ69G5FAV");

    expect(pool.query).toHaveBeenCalledOnce();
    const [sql, params] = pool.query.mock.calls[0];
    expect(sql).toContain("UPDATE outbox_events");
    expect(sql).toContain("delivered = TRUE");
    expect(params[0]).toBe("01ARZ3NDEKTSV4RRFFQ69G5FAV");
  });
});

describe("pendingEvents", () => {
  it("queries undelivered events with limit", async () => {
    const mockRows = [
      {
        event_id: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        aggregate_type: "book",
        aggregate_id: "book-1",
        event_type: "book.created",
        payload: { eventId: "01ARZ3NDEKTSV4RRFFQ69G5FAV" },
        created_at: new Date(),
        delivered: false,
      },
    ];
    const pool = createMockPool();
    pool.query.mockResolvedValue({ rows: mockRows, rowCount: 1 });

    const rows = await pendingEvents(pool as any, 10);

    expect(pool.query).toHaveBeenCalledOnce();
    const [sql, params] = pool.query.mock.calls[0];
    expect(sql).toContain("delivered = FALSE");
    expect(sql).toContain("LIMIT $1");
    expect(params[0]).toBe(10);
    expect(rows).toEqual(mockRows);
  });
});
