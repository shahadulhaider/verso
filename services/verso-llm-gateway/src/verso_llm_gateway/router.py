from __future__ import annotations

from verso_llm_gateway.config import settings

TIER_MAP = {
    "default": lambda: settings.ollama_model,
    "embedding": lambda: settings.ollama_embed_model,
    "moderation": lambda: settings.ollama_model,
}


def resolve_model(model_tier: str | None) -> str:
    tier = model_tier or "default"
    resolver = TIER_MAP.get(tier, TIER_MAP["default"])
    return resolver()
