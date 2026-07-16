from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from opentelemetry import trace
from pydantic import BaseModel, Field
from verso.errors import create_problem, problem_json_response
from verso.logger import create_logger
from verso.otel import init_telemetry

from verso_ai_inference.config import settings
from verso_ai_inference.consumer import catalog_consumer
from verso_ai_inference.embedder import embedder
from verso_ai_inference.vector_store import vector_store


logger = create_logger("verso-ai-inference")
tracer = trace.get_tracer("verso-ai-inference")


@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
    _shutdown_otel = init_telemetry(settings.service_name)

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
    _shutdown_otel()
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
            problem = create_problem(502, "LLM Gateway Unavailable", "Could not generate embedding — LLM gateway unreachable")
            return JSONResponse(content=problem_json_response(problem), status_code=502)

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
