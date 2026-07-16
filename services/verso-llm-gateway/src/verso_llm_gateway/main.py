from __future__ import annotations

import time
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from typing import Literal

import httpx
import structlog
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from pydantic import BaseModel, Field
from ulid import ULID

from verso_llm_gateway.cache import cache_key, llm_cache
from verso_llm_gateway.config import settings
from verso_llm_gateway.providers.ollama import OllamaError, ollama_client
from verso_llm_gateway.registry import prompt_registry
from verso_llm_gateway.router import resolve_model


def _init_telemetry() -> None:
    resource = Resource.create({"service.name": settings.service_name})
    provider = TracerProvider(resource=resource)
    exporter = OTLPSpanExporter(
        endpoint=settings.otel_exporter_otlp_endpoint, insecure=True
    )
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)


logger = structlog.get_logger(service="verso-llm-gateway")
tracer = trace.get_tracer("verso-llm-gateway")


@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
    _init_telemetry()
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.stdlib.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.format_exc_info,
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )

    ollama_client.set_client(httpx.AsyncClient())
    await llm_cache.connect()

    try:
        await prompt_registry.connect()
    except Exception:
        logger.warning("db_connect_failed", detail="Prompt registry unavailable at startup")

    logger.info("started", port=settings.service_port)
    yield

    await ollama_client.client.aclose()
    await llm_cache.close()
    await prompt_registry.close()
    logger.info("shutdown_complete")


app = FastAPI(title="verso-llm-gateway", lifespan=lifespan)


# ── Request / Response models ──────────────────────────────────────────


class CompleteRequest(BaseModel):
    prompt: str
    model_tier: str | None = Field(None, alias="modelTier")
    feature: str = "default"
    max_tokens: int | None = Field(None, alias="maxTokens")
    temperature: float | None = None
    cache: bool = True

    model_config = {"populate_by_name": True}


class EmbedRequest(BaseModel):
    text: str
    model_tier: str | None = Field(None, alias="modelTier")

    model_config = {"populate_by_name": True}


class ModerateRequest(BaseModel):
    content: str
    content_type: str = Field("text", alias="contentType")

    model_config = {"populate_by_name": True}


# ── Endpoints ──────────────────────────────────────────────────────────


@app.get("/health")
async def health() -> dict:
    return {"status": "ok"}


@app.get("/ready")
async def ready() -> JSONResponse:
    checks: dict[str, bool] = {}

    checks["redis"] = await llm_cache.is_healthy()
    checks["ollama"] = await ollama_client.is_healthy()

    try:
        checks["db"] = await prompt_registry.is_healthy()
    except Exception:
        checks["db"] = False

    all_ok = all(checks.values())
    return JSONResponse(
        content={"status": "ready" if all_ok else "degraded", "checks": checks},
        status_code=200 if all_ok else 503,
    )


@app.post("/v1/llm/complete")
async def complete(req: CompleteRequest) -> JSONResponse:
    model_id = resolve_model(req.model_tier)
    start = time.monotonic()

    with tracer.start_as_current_span("llm.complete") as span:
        span.set_attribute("model_id", model_id)
        span.set_attribute("provider", "ollama")
        span.set_attribute("feature", req.feature)

        if req.cache and settings.cache_enabled:
            key = cache_key(req.feature, req.model_tier or "default", req.prompt)
            cached = await llm_cache.get(key)
            if cached is not None:
                latency = int((time.monotonic() - start) * 1000)
                span.set_attribute("cache_hit", True)
                span.set_attribute("latency_ms", latency)
                cached["cacheHit"] = True
                cached["latencyMs"] = latency
                return JSONResponse(content=cached)

        try:
            result = await ollama_client.generate(
                model=model_id,
                prompt=req.prompt,
                max_tokens=req.max_tokens,
                temperature=req.temperature,
            )
        except OllamaError as exc:
            span.set_attribute("error", True)
            return JSONResponse(
                content={"type": f"https://httpstatuses.io/{exc.status_code}", "title": "LLM Error", "status": exc.status_code, "detail": exc.detail},
                status_code=exc.status_code,
            )

        latency = int((time.monotonic() - start) * 1000)
        input_tokens = result.get("prompt_eval_count", 0)
        output_tokens = result.get("eval_count", 0)

        span.set_attribute("input_tokens", input_tokens)
        span.set_attribute("output_tokens", output_tokens)
        span.set_attribute("latency_ms", latency)
        span.set_attribute("cache_hit", False)

        response = {
            "completion": result.get("response", ""),
            "usage": {"inputTokens": input_tokens, "outputTokens": output_tokens},
            "modelId": model_id,
            "cacheHit": False,
            "latencyMs": latency,
        }

        if req.cache and settings.cache_enabled:
            key = cache_key(req.feature, req.model_tier or "default", req.prompt)
            await llm_cache.set(key, response)

        await _log_audit(
            model_id=model_id,
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            latency_ms=latency,
            cache_hit=False,
        )

        return JSONResponse(content=response)


@app.post("/v1/llm/embed")
async def embed(req: EmbedRequest) -> JSONResponse:
    model_id = resolve_model("embedding")
    start = time.monotonic()

    with tracer.start_as_current_span("llm.embed") as span:
        span.set_attribute("model_id", model_id)
        span.set_attribute("provider", "ollama")

        try:
            result = await ollama_client.embed(model=model_id, text=req.text)
        except OllamaError as exc:
            span.set_attribute("error", True)
            return JSONResponse(
                content={"type": f"https://httpstatuses.io/{exc.status_code}", "title": "LLM Error", "status": exc.status_code, "detail": exc.detail},
                status_code=exc.status_code,
            )

        latency = int((time.monotonic() - start) * 1000)
        embeddings = result.get("embeddings", [[]])[0]
        span.set_attribute("latency_ms", latency)

        return JSONResponse(content={
            "embedding": embeddings,
            "dimensions": len(embeddings),
            "modelId": model_id,
        })


MODERATION_PROMPT = (
    "You are a content moderation system. Analyze the following content and respond with ONLY "
    "a JSON object with these fields: verdict (one of: safe, flagged, review), confidence (0.0-1.0), "
    "reason (brief explanation). Content to analyze:\n\n{content}"
)


@app.post("/v1/llm/moderate")
async def moderate(req: ModerateRequest) -> JSONResponse:
    import json as _json

    model_id = resolve_model("moderation")
    start = time.monotonic()

    with tracer.start_as_current_span("llm.moderate") as span:
        span.set_attribute("model_id", model_id)
        span.set_attribute("provider", "ollama")
        span.set_attribute("content_type", req.content_type)

        prompt = MODERATION_PROMPT.format(content=req.content)

        try:
            result = await ollama_client.generate(model=model_id, prompt=prompt)
        except OllamaError as exc:
            span.set_attribute("error", True)
            return JSONResponse(
                content={"type": f"https://httpstatuses.io/{exc.status_code}", "title": "LLM Error", "status": exc.status_code, "detail": exc.detail},
                status_code=exc.status_code,
            )

        latency = int((time.monotonic() - start) * 1000)
        span.set_attribute("latency_ms", latency)

        raw = result.get("response", "")
        try:
            parsed = _json.loads(raw)
            verdict: Literal["safe", "flagged", "review"] = parsed.get("verdict", "review")
            confidence: float = float(parsed.get("confidence", 0.5))
            reason: str = parsed.get("reason", "")
        except (_json.JSONDecodeError, ValueError):
            verdict = "review"
            confidence = 0.0
            reason = f"Model returned unparseable response: {raw[:200]}"

        return JSONResponse(content={
            "verdict": verdict,
            "confidence": confidence,
            "reason": reason,
        })


async def _log_audit(
    *,
    model_id: str,
    input_tokens: int,
    output_tokens: int,
    latency_ms: int,
    cache_hit: bool,
    prompt_id: str | None = None,
) -> None:
    try:
        await prompt_registry.pool.execute(
            "INSERT INTO ai.llm_audit_log (id, prompt_id, model_id, provider, input_tokens, output_tokens, latency_ms, cache_hit) "
            "VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
            str(ULID()),
            prompt_id,
            model_id,
            "ollama",
            input_tokens,
            output_tokens,
            latency_ms,
            cache_hit,
        )
    except Exception:
        logger.warning("audit_log_failed", model_id=model_id)
