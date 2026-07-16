from __future__ import annotations

import json
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient


class TestEndpoints:
    @pytest.fixture()
    def client(self) -> TestClient:
        with (
            patch("verso_ai_inference.main.vector_store") as mock_vs,
            patch("verso_ai_inference.main.embedder") as mock_emb,
            patch("verso_ai_inference.main.catalog_consumer") as mock_consumer,
            patch("verso_ai_inference.main.init_telemetry", return_value=lambda: None),
        ):
            mock_vs.connect = AsyncMock()
            mock_vs.close = AsyncMock()
            mock_vs.is_healthy = AsyncMock(return_value=True)
            mock_vs.find_similar = AsyncMock(return_value=[])
            mock_vs.upsert_embedding = AsyncMock(return_value="01J0000000000000000000TEST")
            mock_vs.has_embedding = AsyncMock(return_value=False)
            mock_vs.write_outbox_event = AsyncMock(return_value="01J00000000000000000OUTBOX")

            mock_emb.set_client = lambda c: None
            mock_emb.close = AsyncMock()

            mock_consumer.start = AsyncMock()
            mock_consumer.stop = AsyncMock()

            from verso_ai_inference.main import app

            with TestClient(app) as tc:
                self._mock_vs = mock_vs
                self._mock_emb = mock_emb
                self._mock_consumer = mock_consumer
                yield tc

    def test_health(self, client: TestClient) -> None:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"

    def test_ready_healthy(self, client: TestClient) -> None:
        resp = client.get("/ready")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ready"
        assert data["checks"]["db"] is True

    def test_ready_degraded(self, client: TestClient) -> None:
        self._mock_vs.is_healthy = AsyncMock(return_value=False)
        resp = client.get("/ready")
        assert resp.status_code == 503
        assert resp.json()["status"] == "degraded"

    def test_embed_success(self, client: TestClient) -> None:
        from verso_ai_inference.embedder import EmbeddingResult

        mock_result = EmbeddingResult(
            embedding=[0.1] * 1024,
            dimensions=1024,
            model_id="nomic-embed-text",
        )
        self._mock_emb.embed = AsyncMock(return_value=mock_result)

        resp = client.post("/v1/ai/embed", json={"text": "Test book title"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["dimensions"] == 1024
        assert data["modelId"] == "nomic-embed-text"
        assert len(data["embedding"]) == 1024

    def test_embed_gateway_unavailable(self, client: TestClient) -> None:
        self._mock_emb.embed = AsyncMock(return_value=None)

        resp = client.post("/v1/ai/embed", json={"text": "Test text"})
        assert resp.status_code == 502
        assert "LLM Gateway Unavailable" in resp.json()["title"]

    def test_similar_success(self, client: TestClient) -> None:
        self._mock_vs.find_similar = AsyncMock(return_value=[
            {"workId": "01J0000000000000000000WORK", "score": 0.95},
            {"workId": "01J0000000000000000001WORK", "score": 0.87},
        ])

        resp = client.post("/v1/ai/similar", json={"workId": "01J0000000000000000SRCWORK"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["workId"] == "01J0000000000000000SRCWORK"
        assert data["count"] == 2
        assert data["results"][0]["score"] == 0.95

    def test_similar_no_results(self, client: TestClient) -> None:
        self._mock_vs.find_similar = AsyncMock(return_value=[])

        resp = client.post("/v1/ai/similar", json={"workId": "01J0000000000000NOEMBWORK"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["count"] == 0
        assert data["results"] == []

    def test_similar_custom_limit(self, client: TestClient) -> None:
        self._mock_vs.find_similar = AsyncMock(return_value=[])

        resp = client.post("/v1/ai/similar", json={"workId": "01J0000000000000000SRCWORK", "limit": 5})
        assert resp.status_code == 200
        self._mock_vs.find_similar.assert_awaited_once_with(
            work_id="01J0000000000000000SRCWORK",
            limit=5,
        )


class TestEmbedder:
    @pytest.fixture()
    def _embedder(self):
        from verso_ai_inference.embedder import Embedder

        e = Embedder()
        e.set_client(MagicMock())
        return e

    def test_embed_result_slots(self) -> None:
        from verso_ai_inference.embedder import EmbeddingResult

        r = EmbeddingResult(embedding=[0.1, 0.2], dimensions=2, model_id="test")
        assert r.embedding == [0.1, 0.2]
        assert r.dimensions == 2
        assert r.model_id == "test"

    def test_embedder_not_initialized(self) -> None:
        from verso_ai_inference.embedder import Embedder

        e = Embedder()
        with pytest.raises(RuntimeError, match="not initialized"):
            _ = e.client


class TestConsumer:
    @pytest.fixture()
    def consumer_deps(self):
        with (
            patch("verso_ai_inference.consumer.vector_store") as mock_vs,
            patch("verso_ai_inference.consumer.embedder") as mock_emb,
        ):
            mock_vs.has_embedding = AsyncMock(return_value=False)
            mock_vs.upsert_embedding = AsyncMock(return_value="01J0TEST")
            mock_vs.write_outbox_event = AsyncMock(return_value="01J0EVT")

            from verso_ai_inference.embedder import EmbeddingResult

            mock_emb.embed = AsyncMock(return_value=EmbeddingResult(
                embedding=[0.1] * 1024,
                dimensions=1024,
                model_id="nomic-embed-text",
            ))
            yield mock_vs, mock_emb

    @pytest.mark.asyncio
    async def test_handle_work_created(self, consumer_deps) -> None:
        mock_vs, mock_emb = consumer_deps
        from verso_ai_inference.consumer import CatalogConsumer

        c = CatalogConsumer()
        await c._handle_work_created({
            "workId": "01JWORK00000000000000000A",
            "title": "The Great Gatsby",
            "description": "A novel about the American dream",
        })

        mock_emb.embed.assert_awaited_once_with("The Great Gatsby. A novel about the American dream")
        mock_vs.upsert_embedding.assert_awaited_once()
        mock_vs.write_outbox_event.assert_awaited_once()

        call_args = mock_vs.write_outbox_event.call_args
        assert call_args.kwargs["event_type"] == "verso.ai.embedding-indexed.v1"

    @pytest.mark.asyncio
    async def test_handle_work_created_deduplication(self, consumer_deps) -> None:
        mock_vs, mock_emb = consumer_deps
        mock_vs.has_embedding = AsyncMock(return_value=True)

        from verso_ai_inference.consumer import CatalogConsumer

        c = CatalogConsumer()
        await c._handle_work_created({
            "workId": "01JWORK00000000000000000B",
            "title": "Already Indexed",
            "description": "This work already has an embedding",
        })

        mock_emb.embed.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_handle_work_created_missing_work_id(self, consumer_deps) -> None:
        mock_vs, mock_emb = consumer_deps
        from verso_ai_inference.consumer import CatalogConsumer

        c = CatalogConsumer()
        await c._handle_work_created({"title": "No ID"})

        mock_emb.embed.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_handle_work_created_embedding_failure(self, consumer_deps) -> None:
        mock_vs, mock_emb = consumer_deps
        mock_emb.embed = AsyncMock(return_value=None)

        from verso_ai_inference.consumer import CatalogConsumer

        c = CatalogConsumer()
        await c._handle_work_created({
            "workId": "01JWORK00000000000000000C",
            "title": "Gateway Down",
            "description": "Cannot reach LLM gateway",
        })

        mock_vs.upsert_embedding.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_handle_edition_published_skip_existing(self, consumer_deps) -> None:
        mock_vs, mock_emb = consumer_deps
        mock_vs.has_embedding = AsyncMock(return_value=True)

        from verso_ai_inference.consumer import CatalogConsumer

        c = CatalogConsumer()
        await c._handle_edition_published({
            "workId": "01JWORK00000000000000000D",
            "title": "Existing",
        })

        mock_emb.embed.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_handle_edition_published_new_embedding(self, consumer_deps) -> None:
        mock_vs, mock_emb = consumer_deps

        from verso_ai_inference.consumer import CatalogConsumer

        c = CatalogConsumer()
        await c._handle_edition_published({
            "workId": "01JWORK00000000000000000E",
            "title": "New Edition",
            "description": "First edition published",
        })

        mock_emb.embed.assert_awaited_once()
        mock_vs.upsert_embedding.assert_awaited_once()


class TestVectorStore:
    def test_pool_not_connected(self) -> None:
        from verso_ai_inference.vector_store import VectorStore

        vs = VectorStore()
        with pytest.raises(RuntimeError, match="not connected"):
            _ = vs.pool


class TestConfig:
    def test_defaults(self) -> None:
        from verso_ai_inference.config import Settings

        s = Settings()
        assert s.service_name == "verso-ai-inference-service"
        assert s.service_port == 8012
        assert s.kafka_consumer_group == "verso-ai-inference"
        assert "verso-llm-gateway" in s.llm_gateway_url
