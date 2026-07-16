from __future__ import annotations

import asyncpg

from verso_llm_gateway.config import settings


class PromptRegistry:
    def __init__(self) -> None:
        self._pool: asyncpg.Pool | None = None

    @property
    def pool(self) -> asyncpg.Pool:
        if self._pool is None:
            raise RuntimeError("PromptRegistry not initialized — call connect first")
        return self._pool

    async def connect(self) -> None:
        dsn = settings.database_url.replace("postgres://", "postgresql://", 1)
        self._pool = await asyncpg.create_pool(dsn, min_size=2, max_size=10)

    async def close(self) -> None:
        if self._pool is not None:
            await self._pool.close()

    async def get_active_prompt(self, prompt_id: str) -> asyncpg.Record | None:
        return await self.pool.fetchrow(
            "SELECT * FROM ai.prompt_registry WHERE prompt_id = $1 AND is_active = TRUE ORDER BY version DESC LIMIT 1",
            prompt_id,
        )

    async def is_healthy(self) -> bool:
        try:
            await self.pool.fetchval("SELECT 1")
            return True
        except Exception:
            return False


prompt_registry = PromptRegistry()
