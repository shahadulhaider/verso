from __future__ import annotations

import asyncio
import json

import structlog
from aiokafka import AIOKafkaConsumer

from verso_ai_inference.config import settings
from verso_ai_inference.embedder import embedder
from verso_ai_inference.vector_store import vector_store

logger = structlog.get_logger(component="consumer")

TOPICS = [
    "verso.catalog.work-created.v1",
    "verso.catalog.edition-published.v1",
]


class CatalogConsumer:
    """Kafka consumer that processes catalog events to generate embeddings."""

    def __init__(self) -> None:
        self._consumer: AIOKafkaConsumer | None = None
        self._task: asyncio.Task[None] | None = None

    async def start(self) -> None:
        self._consumer = AIOKafkaConsumer(
            *TOPICS,
            bootstrap_servers=settings.kafka_brokers,
            group_id=settings.kafka_consumer_group,
            auto_offset_reset="earliest",
            enable_auto_commit=True,
            value_deserializer=lambda v: json.loads(v.decode("utf-8")) if v else None,
        )
        await self._consumer.start()
        self._task = asyncio.create_task(self._consume_loop())
        logger.info("consumer_started", topics=TOPICS)

    async def stop(self) -> None:
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        if self._consumer:
            await self._consumer.stop()
        logger.info("consumer_stopped")

    async def _consume_loop(self) -> None:
        if self._consumer is None:
            return
        try:
            async for msg in self._consumer:
                try:
                    await self._handle_message(msg.topic, msg.value)
                except Exception:
                    logger.exception("message_processing_failed", topic=msg.topic)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.exception("consumer_loop_error")

    async def _handle_message(self, topic: str, value: dict | None) -> None:
        if value is None:
            return

        payload = value.get("payload", value)

        if topic == "verso.catalog.work-created.v1":
            await self._handle_work_created(payload)
        elif topic == "verso.catalog.edition-published.v1":
            await self._handle_edition_published(payload)

    async def _handle_work_created(self, payload: dict) -> None:
        work_id = payload.get("workId") or payload.get("work_id")
        title = payload.get("title", "")
        description = payload.get("description", "")

        if not work_id:
            logger.warning("work_created_missing_work_id", payload=payload)
            return

        if await vector_store.has_embedding(work_id):
            logger.info("embedding_already_exists", work_id=work_id)
            return

        text = f"{title}. {description}".strip()
        if not text or text == ".":
            logger.warning("work_created_empty_text", work_id=work_id)
            return

        result = await embedder.embed(text)
        if result is None:
            logger.warning("embedding_generation_failed", work_id=work_id)
            return

        await vector_store.upsert_embedding(
            work_id=work_id,
            embedding=result.embedding,
            embedding_model=result.model_id,
        )

        await vector_store.write_outbox_event(
            aggregate_type="WorkEmbedding",
            aggregate_id=work_id,
            event_type="verso.ai.embedding-indexed.v1",
            payload={
                "workId": work_id,
                "embeddingModel": result.model_id,
                "dimensions": result.dimensions,
            },
        )

        logger.info("work_embedding_created", work_id=work_id, model=result.model_id)

    async def _handle_edition_published(self, payload: dict) -> None:
        work_id = payload.get("workId") or payload.get("work_id")
        if not work_id:
            logger.warning("edition_published_missing_work_id", payload=payload)
            return

        if await vector_store.has_embedding(work_id):
            logger.info("edition_published_embedding_exists", work_id=work_id)
            return

        title = payload.get("title", "")
        description = payload.get("description", "")
        text = f"{title}. {description}".strip()

        if not text or text == ".":
            return

        result = await embedder.embed(text)
        if result is None:
            return

        await vector_store.upsert_embedding(
            work_id=work_id,
            embedding=result.embedding,
            embedding_model=result.model_id,
        )

        await vector_store.write_outbox_event(
            aggregate_type="WorkEmbedding",
            aggregate_id=work_id,
            event_type="verso.ai.embedding-indexed.v1",
            payload={
                "workId": work_id,
                "embeddingModel": result.model_id,
                "dimensions": result.dimensions,
            },
        )


catalog_consumer = CatalogConsumer()
