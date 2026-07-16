import type { FastifyInstance } from "fastify";
import type pg from "pg";
import type { Redis } from "ioredis";
import { requireAuth } from "../middleware/auth.js";

interface TimelineDeps {
  pool: pg.Pool;
  redis: Redis;
}

const SCORE_BOOST: Record<string, number> = {
  reviewed: 1.5,
  followed: 1.2,
};

const DEFAULT_LIMIT = 20;
const MAX_LIMIT = 100;

export async function timelineRoutes(
  app: FastifyInstance,
  { pool, redis }: TimelineDeps,
): Promise<void> {
  app.get("/v1/feed/timeline", { preHandler: [requireAuth] }, async (request, reply) => {
    const { userId } = request as typeof request & { userId: string };
    const query = request.query as { mode?: string; cursor?: string; limit?: string };
    const mode = query.mode === "algorithmic" ? "algorithmic" : "chronological";
    const cursorScore = query.cursor ? parseFloat(query.cursor) : Infinity;
    const limit = Math.min(Math.max(parseInt(query.limit ?? "", 10) || DEFAULT_LIMIT, 1), MAX_LIMIT);

    let items = await readFromRedis(redis, userId, cursorScore, limit, mode);

    if (items.length === 0) {
      items = await readFromPg(pool, userId, cursorScore, limit, mode);
    }

    const nextCursor = items.length > 0 ? items[items.length - 1].score.toString() : null;

    return reply.send({
      items,
      nextCursor,
      mode,
    });
  });
}

interface TimelineItem {
  activityId: string;
  actorId: string;
  verb: string;
  objectType: string;
  objectId: string;
  metadata: Record<string, unknown>;
  score: number;
  createdAt: string;
}

async function readFromRedis(
  redis: Redis,
  userId: string,
  cursorScore: number,
  limit: number,
  mode: string,
): Promise<TimelineItem[]> {
  const key = `feed:timeline:${userId}`;
  const maxScore = cursorScore === Infinity ? "+inf" : `(${cursorScore}`;
  const rawEntries = await redis.zrevrangebyscore(key, maxScore, "-inf", "WITHSCORES", "LIMIT", 0, limit * 2);

  if (rawEntries.length === 0) return [];

  const activityScores: Array<{ activityId: string; score: number }> = [];
  for (let i = 0; i < rawEntries.length; i += 2) {
    activityScores.push({
      activityId: rawEntries[i],
      score: parseFloat(rawEntries[i + 1]),
    });
  }

  return enrichAndRank(activityScores, mode, limit);
}

async function readFromPg(
  pool: pg.Pool,
  userId: string,
  cursorScore: number,
  limit: number,
  mode: string,
): Promise<TimelineItem[]> {
  const cursorCondition = cursorScore === Infinity ? "" : "AND te.score < $3";
  const params: unknown[] = [userId, limit * 2];
  if (cursorScore !== Infinity) params.push(cursorScore);

  const result = await pool.query(
    `SELECT te.activity_id, te.score, a.actor_id, a.verb, a.object_type, a.object_id, a.metadata, a.created_at
     FROM feed.timeline_entry te
     JOIN feed.activity a ON a.id = te.activity_id
     WHERE te.user_id = $1 ${cursorCondition}
     ORDER BY te.score DESC
     LIMIT $2`,
    params,
  );

  const items: TimelineItem[] = result.rows.map((row: Record<string, unknown>) => ({
    activityId: (row.activity_id as string).trim(),
    actorId: (row.actor_id as string).trim(),
    verb: (row.verb as string).trim(),
    objectType: (row.object_type as string).trim(),
    objectId: (row.object_id as string).trim(),
    metadata: row.metadata as Record<string, unknown>,
    score: parseFloat(row.score as string),
    createdAt: (row.created_at as Date).toISOString(),
  }));

  if (mode === "algorithmic") {
    return applyAlgorithmicRanking(items).slice(0, limit);
  }

  return items.slice(0, limit);
}

function applyAlgorithmicRanking(items: TimelineItem[]): TimelineItem[] {
  return items
    .map((item) => ({
      ...item,
      score: item.score * (SCORE_BOOST[item.verb] ?? 1.0),
    }))
    .sort((a, b) => b.score - a.score);
}

async function enrichAndRank(
  activityScores: Array<{ activityId: string; score: number }>,
  mode: string,
  limit: number,
): Promise<TimelineItem[]> {
  const items: TimelineItem[] = activityScores.map((entry) => ({
    activityId: entry.activityId,
    actorId: "",
    verb: "",
    objectType: "",
    objectId: "",
    metadata: {},
    score: entry.score,
    createdAt: new Date(entry.score * 1000).toISOString(),
  }));

  if (mode === "algorithmic") {
    return applyAlgorithmicRanking(items).slice(0, limit);
  }

  return items.slice(0, limit);
}
