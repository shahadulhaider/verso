from __future__ import annotations

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    service_name: str = "verso-llm-gateway"
    service_port: int = 8011

    database_url: str = "postgres://verso:verso_dev@localhost:5432/verso?options=-c%20search_path%3Dai"
    redis_url: str = "redis://localhost:6379/1"

    ollama_url: str = "http://ollama:11434"
    ollama_model: str = "llama3.2"
    ollama_embed_model: str = "nomic-embed-text"
    ollama_timeout_seconds: float = 120.0

    cache_ttl_seconds: int = 3600
    cache_enabled: bool = True

    otel_exporter_otlp_endpoint: str = "http://localhost:4317"

    model_config = {"env_prefix": "", "case_sensitive": False}


settings = Settings()
