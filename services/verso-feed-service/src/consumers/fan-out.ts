import type pg from "pg";
import type { Redis } from "ioredis";
import type CircuitBreaker from "opossum";
import { ulid } from "ulid";
import type { FollowersResponse } from "../lib/social-client.js";

export interface Activity {
  id: string;
  actorId: string;
  verb: string;
  objectType: string;
  objectId: string;
  metadata: Record<string, unknown>;
  createdAt: Date;
}

export interface FanOutDeps {
  pool: pg.Pool;
  redis: Redis;
  socialBreaker: CircuitBreaker<[string], FollowersResponse>;
  cacheTtl: number;
  logger: { info: (...args: unknown[]) => void; error: (...args: unknown[]) => void; warn: (...args: unknown[]) => void };
}

export async function createActivity(
  deps: FanOutDeps,
  actorId: string,
  verb: string,
  objectType: string,
  objectId: string,
  metadata: Record<string, unknown> = {},
): Promise<Activity> {
  const activity: Activity = {
    id: ulid(),
    actorId,
    verb,
    objectType,
    objectId,
    metadata,
    createdAt: new Date(),
  };

  await deps.pool.query(
    `INSERT INTO feed.activity (id, actor_id, verb, object_type, object_id, metadata, created_at)
     VALUES ($1, $2, $3, $4, $5, $6, $7)`,
    [activity.id, activity.actorId, activity.verb, activity.objectType, activity.objectId, JSON.stringify(activity.metadata), activity.createdAt],
  );

  const outboxId = ulid();
  await deps.pool.query(
    `INSERT INTO feed.outbox_events (id, aggregate_type, aggregate_id, type, payload, created_at)
     VALUES ($1, $2, $3, $4, $5, $6)`,
    [outboxId, "activity", activity.id, "verso.feed.activity-created.v1", JSON.stringify(activity), activity.createdAt],
  );

  await fanOutToFollowers(deps, activity);

  return activity;
}

async function fanOutToFollowers(deps: FanOutDeps, activity: Activity): Promise<void> {
  let followers: FollowersResponse;
  try {
    followers = await deps.socialBreaker.fire(activity.actorId);
  } catch (err) {
    deps.logger.warn({ err, actorId: activity.actorId }, "Social service unavailable, skipping fan-out (activity saved, can back-fill)");
    return;
  }

  if (!followers.userIds || followers.userIds.length === 0) return;

  const score = activity.createdAt.getTime() / 1000;

  const pgValues: string[] = [];
  const pgParams: unknown[] = [];
  let paramIdx = 1;

  for (const userId of followers.userIds) {
    const entryId = ulid();
    pgValues.push(`($${paramIdx}, $${paramIdx + 1}, $${paramIdx + 2}, $${paramIdx + 3})`);
    pgParams.push(entryId, userId, activity.id, score);
    paramIdx += 4;
  }

  if (pgValues.length > 0) {
    await deps.pool.query(
      `INSERT INTO feed.timeline_entry (id, user_id, activity_id, score) VALUES ${pgValues.join(", ")}`,
      pgParams,
    );
  }

  const pipeline = deps.redis.pipeline();
  const SEVEN_DAYS = deps.cacheTtl;
  for (const userId of followers.userIds) {
    const key = `feed:timeline:${userId}`;
    pipeline.zadd(key, score.toString(), activity.id);
    pipeline.expire(key, SEVEN_DAYS);
  }
  await pipeline.exec();
}
