from __future__ import annotations

import hashlib
import json

import redis.asyncio as redis

from verso_llm_gateway.config import settings


def cache_key(feature: str, model_tier: str, prompt_text: str) -> str:
    raw = f"{feature}:{model_tier}:{prompt_text}"
    digest = hashlib.sha256(raw.encode()).hexdigest()
    return f"llm:cache:{digest}"


class LLMCache:
    def __init__(self) -> None:
        self._redis: redis.Redis | None = None

    @property
    def redis(self) -> redis.Redis:
        if self._redis is None:
            raise RuntimeError("LLMCache not initialized — call connect first")
        return self._redis

    async def connect(self) -> None:
        self._redis = redis.from_url(settings.redis_url, decode_responses=True)

    async def close(self) -> None:
        if self._redis is not None:
            await self._redis.aclose()

    async def get(self, key: str) -> dict | None:
        raw = await self.redis.get(key)
        if raw is None:
            return None
        return json.loads(raw)

    async def set(self, key: str, value: dict, ttl: int | None = None) -> None:
        ttl = ttl or settings.cache_ttl_seconds
        await self.redis.set(key, json.dumps(value), ex=ttl)

    async def is_healthy(self) -> bool:
        try:
            return await self.redis.ping()
        except Exception:
            return False


llm_cache = LLMCache()
