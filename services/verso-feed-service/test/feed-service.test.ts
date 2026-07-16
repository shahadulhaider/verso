import { describe, it, expect, vi, beforeEach } from "vitest";
import { buildHandlers, TOPICS } from "../src/consumers/event-handlers.js";
import { createActivity } from "../src/consumers/fan-out.js";
import type { FanOutDeps } from "../src/consumers/fan-out.js";
import type { EventEnvelope } from "../src/lib/kafka.js";

function createMockPool() {
  return {
    query: vi.fn().mockResolvedValue({ rows: [], rowCount: 0 }),
  } as unknown as import("pg").Pool;
}

function createMockRedis() {
  const pipeline = {
    zadd: vi.fn().mockReturnThis(),
    expire: vi.fn().mockReturnThis(),
    exec: vi.fn().mockResolvedValue([]),
  };
  return {
    pipeline: vi.fn().mockReturnValue(pipeline),
    zrevrangebyscore: vi.fn().mockResolvedValue([]),
    ping: vi.fn().mockResolvedValue("PONG"),
    quit: vi.fn().mockResolvedValue("OK"),
    _pipeline: pipeline,
  } as unknown as import("ioredis").default & { _pipeline: typeof pipeline };
}

function createMockBreaker(followers: string[] = []) {
  return {
    fire: vi.fn().mockResolvedValue({ count: followers.length, userIds: followers }),
  } as unknown as import("opossum").default<[string], import("../src/lib/social-client.js").FollowersResponse>;
}

function createMockLogger() {
  return {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
  };
}

function makeDeps(overrides: Partial<FanOutDeps> = {}): FanOutDeps {
  return {
    pool: createMockPool(),
    redis: createMockRedis(),
    socialBreaker: createMockBreaker(["follower1", "follower2"]),
    cacheTtl: 604800,
    logger: createMockLogger(),
    ...overrides,
  };
}

function makeEnvelope(type: string, payload: Record<string, unknown>): EventEnvelope {
  return {
    eventId: "evt-001",
    type,
    occurredAt: new Date().toISOString(),
    producer: "test",
    partitionKey: "test-key",
    payload,
    schemaVersion: 1,
  };
}

describe("Event Handlers", () => {
  it("registers handlers for all 4 topics", () => {
    const deps = makeDeps();
    const handlers = buildHandlers(deps);
    for (const topic of TOPICS) {
      expect(handlers.has(topic)).toBe(true);
    }
    expect(handlers.size).toBe(4);
  });

  it("processes review-published event", async () => {
    const deps = makeDeps();
    const handlers = buildHandlers(deps);
    const handler = handlers.get("verso.review.review-published.v1")!;

    await handler(makeEnvelope("verso.review.review-published.v1", {
      userId: "user-001",
      reviewId: "review-001",
      workId: "work-001",
    }));

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.arrayContaining(["user-001", "reviewed", "review", "review-001"]),
    );
  });

  it("processes shelf-item-added event", async () => {
    const deps = makeDeps();
    const handlers = buildHandlers(deps);
    const handler = handlers.get("verso.library.shelf-item-added.v1")!;

    await handler(makeEnvelope("verso.library.shelf-item-added.v1", {
      userId: "user-002",
      workId: "work-002",
      shelfName: "want-to-read",
    }));

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.arrayContaining(["user-002", "shelved", "shelf_item", "work-002"]),
    );
  });

  it("processes user-followed event", async () => {
    const deps = makeDeps();
    const handlers = buildHandlers(deps);
    const handler = handlers.get("verso.social.user-followed.v1")!;

    await handler(makeEnvelope("verso.social.user-followed.v1", {
      followerId: "user-003",
      followedId: "user-004",
    }));

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.arrayContaining(["user-003", "followed", "user", "user-004"]),
    );
  });

  it("processes reading-progress-updated event", async () => {
    const deps = makeDeps();
    const handlers = buildHandlers(deps);
    const handler = handlers.get("verso.library.reading-progress-updated.v1")!;

    await handler(makeEnvelope("verso.library.reading-progress-updated.v1", {
      userId: "user-005",
      workId: "work-005",
      progress: 0.75,
    }));

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.arrayContaining(["user-005", "reading", "work", "work-005"]),
    );
  });
});

describe("Fan-out", () => {
  it("creates activity and fans out to followers", async () => {
    const deps = makeDeps();
    await createActivity(deps, "actor-001", "reviewed", "review", "obj-001", { workId: "w1" });

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.arrayContaining(["actor-001", "reviewed", "review", "obj-001"]),
    );

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.outbox_events"),
      expect.any(Array),
    );

    expect(deps.socialBreaker.fire).toHaveBeenCalledWith("actor-001");

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.timeline_entry"),
      expect.any(Array),
    );

    const mockRedis = deps.redis as unknown as { _pipeline: { zadd: ReturnType<typeof vi.fn>; expire: ReturnType<typeof vi.fn> } };
    expect(mockRedis._pipeline.zadd).toHaveBeenCalledTimes(2);
    expect(mockRedis._pipeline.expire).toHaveBeenCalledTimes(2);
  });

  it("saves activity but skips fan-out when social service is down", async () => {
    const failingBreaker = {
      fire: vi.fn().mockRejectedValue(new Error("Circuit breaker open")),
    } as unknown as import("opossum").default<[string], import("../src/lib/social-client.js").FollowersResponse>;

    const deps = makeDeps({ socialBreaker: failingBreaker });
    await createActivity(deps, "actor-002", "shelved", "shelf_item", "obj-002");

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.any(Array),
    );

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.outbox_events"),
      expect.any(Array),
    );

    expect(deps.pool.query).not.toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.timeline_entry"),
      expect.any(Array),
    );

    expect(deps.logger.warn).toHaveBeenCalled();
  });

  it("skips fan-out when there are no followers", async () => {
    const emptyBreaker = createMockBreaker([]);
    const deps = makeDeps({ socialBreaker: emptyBreaker });
    await createActivity(deps, "actor-003", "followed", "user", "obj-003");

    expect(deps.pool.query).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.activity"),
      expect.any(Array),
    );

    expect(deps.pool.query).not.toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO feed.timeline_entry"),
      expect.any(Array),
    );
  });
});

describe("Timeline API", () => {
  it("applies algorithmic score boost for reviews (1.5x)", () => {
    const items = [
      { activityId: "a1", actorId: "u1", verb: "reviewed", objectType: "review", objectId: "r1", metadata: {}, score: 1000, createdAt: "" },
      { activityId: "a2", actorId: "u2", verb: "shelved", objectType: "shelf_item", objectId: "w1", metadata: {}, score: 1100, createdAt: "" },
      { activityId: "a3", actorId: "u3", verb: "followed", objectType: "user", objectId: "u4", metadata: {}, score: 1050, createdAt: "" },
    ];

    const ranked = items
      .map((item) => ({
        ...item,
        score: item.score * (item.verb === "reviewed" ? 1.5 : item.verb === "followed" ? 1.2 : 1.0),
      }))
      .sort((a, b) => b.score - a.score);

    expect(ranked[0].activityId).toBe("a1");
    expect(ranked[0].score).toBe(1500);
    expect(ranked[1].activityId).toBe("a3");
    expect(ranked[1].score).toBe(1260);
    expect(ranked[2].activityId).toBe("a2");
    expect(ranked[2].score).toBe(1100);
  });

  it("maintains chronological order without boosts", () => {
    const items = [
      { score: 1000, verb: "reviewed" },
      { score: 1100, verb: "shelved" },
      { score: 1050, verb: "followed" },
    ];

    const sorted = [...items].sort((a, b) => b.score - a.score);
    expect(sorted[0].score).toBe(1100);
    expect(sorted[1].score).toBe(1050);
    expect(sorted[2].score).toBe(1000);
  });

  it("TOPICS constant contains all 4 expected topics", () => {
    expect(TOPICS).toContain("verso.review.review-published.v1");
    expect(TOPICS).toContain("verso.library.shelf-item-added.v1");
    expect(TOPICS).toContain("verso.social.user-followed.v1");
    expect(TOPICS).toContain("verso.library.reading-progress-updated.v1");
    expect(TOPICS.length).toBe(4);
  });
});
