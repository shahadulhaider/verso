from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

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

from verso_ai_inference.config import settings
from verso_ai_inference.consumer import catalog_consumer
from verso_ai_inference.embedder import embedder
from verso_ai_inference.vector_store import vector_store


def _init_telemetry() -> None:
    resource = Resource.create({"service.name": settings.service_name})
    provider = TracerProvider(resource=resource)
    exporter = OTLPSpanExporter(
        endpoint=settings.otel_exporter_otlp_endpoint, insecure=True
    )
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)


logger = structlog.get_logger(service="verso-ai-inference")
tracer = trace.get_tracer("verso-ai-inference")


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

    embedder.set_client(httpx.AsyncClient())
    await vector_store.connect()

    try:
        await catalog_consumer.start()
    except Exception:
        logger.warning("kafka_consumer_start_failed", detail="Consumer unavailable at startup")

    logger.info("started", port=settings.service_port)
    yield

    await catalog_consumer.stop()
    await embedder.close()
    await vector_store.close()
    logger.info("shutdown_complete")


app = FastAPI(title="verso-ai-inference-service", lifespan=lifespan)


# ── Request / Response models ──────────────────────────────────────────


class EmbedRequest(BaseModel):
    text: str


class SimilarRequest(BaseModel):
    work_id: str = Field(..., alias="workId")
    limit: int = 10

    model_config = {"populate_by_name": True}


# ── Endpoints ──────────────────────────────────────────────────────────


@app.get("/health")
async def health() -> dict:
    return {"status": "ok"}


@app.get("/ready")
async def ready() -> JSONResponse:
    checks: dict[str, bool] = {}

    try:
        checks["db"] = await vector_store.is_healthy()
    except Exception:
        checks["db"] = False

    all_ok = all(checks.values())
    return JSONResponse(
        content={"status": "ready" if all_ok else "degraded", "checks": checks},
        status_code=200 if all_ok else 503,
    )


@app.post("/v1/ai/embed")
async def embed(req: EmbedRequest) -> JSONResponse:
    with tracer.start_as_current_span("ai.embed") as span:
        span.set_attribute("text_length", len(req.text))

        result = await embedder.embed(req.text)
        if result is None:
            return JSONResponse(
                content={
                    "type": "https://httpstatuses.io/502",
                    "title": "LLM Gateway Unavailable",
                    "status": 502,
                    "detail": "Could not generate embedding — LLM gateway unreachable",
                },
                status_code=502,
            )

        span.set_attribute("dimensions", result.dimensions)
        span.set_attribute("model_id", result.model_id)

        return JSONResponse(content={
            "embedding": result.embedding,
            "dimensions": result.dimensions,
            "modelId": result.model_id,
        })


@app.post("/v1/ai/similar")
async def similar(req: SimilarRequest) -> JSONResponse:
    with tracer.start_as_current_span("ai.similar") as span:
        span.set_attribute("work_id", req.work_id)
        span.set_attribute("limit", req.limit)

        results = await vector_store.find_similar(
            work_id=req.work_id,
            limit=req.limit,
        )

        span.set_attribute("result_count", len(results))

        return JSONResponse(content={
            "workId": req.work_id,
            "results": results,
            "count": len(results),
        })
