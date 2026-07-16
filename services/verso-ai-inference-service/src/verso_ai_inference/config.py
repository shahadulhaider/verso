from __future__ import annotations

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    service_name: str = "verso-ai-inference-service"
    service_port: int = 8012

    database_url: str = "postgres://verso:verso_dev@localhost:5432/verso?options=-c%20search_path%3Dai"

    llm_gateway_url: str = "http://verso-llm-gateway:8011"
    llm_gateway_timeout_seconds: float = 30.0

    kafka_brokers: str = "localhost:9092"
    kafka_consumer_group: str = "verso-ai-inference"

    otel_exporter_otlp_endpoint: str = "http://localhost:4317"

    model_config = {"env_prefix": "", "case_sensitive": False}


settings = Settings()
