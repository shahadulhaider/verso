from __future__ import annotations

from typing import Any

import asyncpg
import structlog
from pgvector.asyncpg import register_vector
from ulid import ULID

from verso_ai_inference.config import settings

logger = structlog.get_logger(component="vector_store")


class VectorStore:
    def __init__(self) -> None:
        self._pool: asyncpg.Pool | None = None

    async def connect(self) -> None:
        self._pool = await asyncpg.create_pool(
            settings.database_url,
            min_size=2,
            max_size=10,
            init=_init_connection,
        )
        logger.info("vector_store_connected")

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()
            logger.info("vector_store_closed")

    @property
    def pool(self) -> asyncpg.Pool:
        if self._pool is None:
            msg = "VectorStore not connected"
            raise RuntimeError(msg)
        return self._pool

    async def is_healthy(self) -> bool:
        try:
            async with self.pool.acquire() as conn:
                await conn.fetchval("SELECT 1")
            return True
        except Exception:
            return False

    async def upsert_embedding(
        self,
        work_id: str,
        embedding: list[float],
        embedding_model: str,
    ) -> str:
        row_id = str(ULID())
        await self.pool.execute(
            """
            INSERT INTO ai.work_embedding (id, work_id, embedding, embedding_model)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (work_id) DO UPDATE
              SET embedding = $3,
                  embedding_model = $4,
                  updated_at = NOW()
            """,
            row_id,
            work_id,
            embedding,
            embedding_model,
        )
        logger.info("embedding_upserted", work_id=work_id, model=embedding_model)
        return row_id

    async def has_embedding(self, work_id: str) -> bool:
        count = await self.pool.fetchval(
            "SELECT COUNT(*) FROM ai.work_embedding WHERE work_id = $1",
            work_id,
        )
        return count > 0

    async def find_similar(
        self,
        work_id: str,
        limit: int = 10,
    ) -> list[dict[str, Any]]:
        row = await self.pool.fetchrow(
            "SELECT embedding FROM ai.work_embedding WHERE work_id = $1",
            work_id,
        )
        if row is None:
            return []

        embedding = row["embedding"]
        rows = await self.pool.fetch(
            """
            SELECT work_id, 1 - (embedding <=> $1) AS score
            FROM ai.work_embedding
            WHERE work_id != $2
            ORDER BY embedding <=> $1
            LIMIT $3
            """,
            embedding,
            work_id,
            limit,
        )
        return [{"workId": r["work_id"], "score": float(r["score"])} for r in rows]

    async def write_outbox_event(
        self,
        aggregate_type: str,
        aggregate_id: str,
        event_type: str,
        payload: dict[str, Any],
    ) -> str:
        import json

        event_id = str(ULID())
        await self.pool.execute(
            """
            INSERT INTO ai.outbox_events (id, aggregate_type, aggregate_id, type, payload)
            VALUES ($1, $2, $3, $4, $5)
            """,
            event_id,
            aggregate_type,
            aggregate_id,
            event_type,
            json.dumps(payload),
        )
        return event_id


async def _init_connection(conn: asyncpg.Connection) -> None:
    await register_vector(conn)


vector_store = VectorStore()
