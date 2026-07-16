import type { EventEnvelope, MessageHandler } from "../lib/kafka.js";
import type { FanOutDeps } from "./fan-out.js";
import { createActivity } from "./fan-out.js";

export const TOPICS = [
  "verso.review.review-published.v1",
  "verso.library.shelf-item-added.v1",
  "verso.social.user-followed.v1",
  "verso.library.reading-progress-updated.v1",
] as const;

function extractPayload(envelope: EventEnvelope): Record<string, unknown> {
  return typeof envelope.payload === "string"
    ? JSON.parse(envelope.payload as string)
    : envelope.payload;
}

export function buildHandlers(deps: FanOutDeps): Map<string, MessageHandler> {
  const handlers = new Map<string, MessageHandler>();

  handlers.set("verso.review.review-published.v1", async (envelope: EventEnvelope) => {
    const payload = extractPayload(envelope);
    const actorId = payload.userId as string ?? payload.reviewerId as string;
    const reviewId = payload.reviewId as string ?? payload.id as string;
    await createActivity(deps, actorId, "reviewed", "review", reviewId, { workId: payload.workId });
    deps.logger.info({ actorId, reviewId }, "Processed review-published");
  });

  handlers.set("verso.library.shelf-item-added.v1", async (envelope: EventEnvelope) => {
    const payload = extractPayload(envelope);
    const actorId = payload.userId as string;
    const workId = payload.workId as string;
    await createActivity(deps, actorId, "shelved", "shelf_item", workId, { shelfName: payload.shelfName });
    deps.logger.info({ actorId, workId }, "Processed shelf-item-added");
  });

  handlers.set("verso.social.user-followed.v1", async (envelope: EventEnvelope) => {
    const payload = extractPayload(envelope);
    const actorId = payload.followerId as string;
    const followeeId = payload.followedId as string ?? payload.followeeId as string;
    await createActivity(deps, actorId, "followed", "user", followeeId);
    deps.logger.info({ actorId, followeeId }, "Processed user-followed");
  });

  handlers.set("verso.library.reading-progress-updated.v1", async (envelope: EventEnvelope) => {
    const payload = extractPayload(envelope);
    const actorId = payload.userId as string;
    const workId = payload.workId as string;
    await createActivity(deps, actorId, "reading", "work", workId, { progress: payload.progress });
    deps.logger.info({ actorId, workId }, "Processed reading-progress-updated");
  });

  return handlers;
}
