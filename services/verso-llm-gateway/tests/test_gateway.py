from __future__ import annotations

import hashlib
import json
from unittest.mock import AsyncMock, patch

import httpx
import pytest
from fastapi.testclient import TestClient

from verso_llm_gateway.cache import cache_key
from verso_llm_gateway.router import resolve_model


class TestCacheKey:
    def test_deterministic(self) -> None:
        k1 = cache_key("search", "default", "hello world")
        k2 = cache_key("search", "default", "hello world")
        assert k1 == k2

    def test_different_inputs_different_keys(self) -> None:
        k1 = cache_key("search", "default", "hello")
        k2 = cache_key("search", "default", "goodbye")
        assert k1 != k2

    def test_sha256_format(self) -> None:
        key = cache_key("feat", "tier", "prompt")
        assert key.startswith("llm:cache:")
        hex_part = key.removeprefix("llm:cache:")
        assert len(hex_part) == 64

    def test_feature_matters(self) -> None:
        k1 = cache_key("search", "default", "hello")
        k2 = cache_key("review", "default", "hello")
        assert k1 != k2

    def test_tier_matters(self) -> None:
        k1 = cache_key("search", "default", "hello")
        k2 = cache_key("search", "embedding", "hello")
        assert k1 != k2


class TestRouter:
    def test_default_tier(self) -> None:
        model = resolve_model(None)
        assert model == "llama3.2"

    def test_explicit_default(self) -> None:
        model = resolve_model("default")
        assert model == "llama3.2"

    def test_embedding_tier(self) -> None:
        model = resolve_model("embedding")
        assert model == "nomic-embed-text"

    def test_moderation_tier(self) -> None:
        model = resolve_model("moderation")
        assert model == "llama3.2"

    def test_unknown_tier_falls_back(self) -> None:
        model = resolve_model("unknown_tier")
        assert model == "llama3.2"


class TestEndpoints:
    @pytest.fixture()
    def client(self) -> TestClient:
        with (
            patch("verso_llm_gateway.main.llm_cache") as mock_cache,
            patch("verso_llm_gateway.main.ollama_client") as mock_ollama,
            patch("verso_llm_gateway.main.prompt_registry") as mock_registry,
            patch("verso_llm_gateway.main._init_telemetry"),
        ):
            mock_cache.connect = AsyncMock()
            mock_cache.close = AsyncMock()
            mock_cache.is_healthy = AsyncMock(return_value=True)
            mock_cache.get = AsyncMock(return_value=None)
            mock_cache.set = AsyncMock()

            mock_ollama.set_client = lambda c: None
            mock_ollama.client = AsyncMock()
            mock_ollama.client.aclose = AsyncMock()
            mock_ollama.is_healthy = AsyncMock(return_value=True)

            mock_registry.connect = AsyncMock()
            mock_registry.close = AsyncMock()
            mock_registry.is_healthy = AsyncMock(return_value=True)
            mock_registry.pool = AsyncMock()
            mock_registry.pool.execute = AsyncMock()

            from verso_llm_gateway.main import app

            with TestClient(app) as tc:
                self._mock_cache = mock_cache
                self._mock_ollama = mock_ollama
                self._mock_registry = mock_registry
                yield tc

    def test_health(self, client: TestClient) -> None:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"

    def test_ready_all_healthy(self, client: TestClient) -> None:
        resp = client.get("/ready")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ready"
        assert data["checks"]["redis"] is True
        assert data["checks"]["ollama"] is True

    def test_ready_degraded(self, client: TestClient) -> None:
        self._mock_cache.is_healthy = AsyncMock(return_value=False)
        resp = client.get("/ready")
        assert resp.status_code == 503
        assert resp.json()["status"] == "degraded"

    def test_complete_success(self, client: TestClient) -> None:
        self._mock_ollama.generate = AsyncMock(return_value={
            "response": "Hello from LLM",
            "prompt_eval_count": 10,
            "eval_count": 5,
        })

        resp = client.post("/v1/llm/complete", json={
            "prompt": "Say hello",
            "feature": "test",
        })

        assert resp.status_code == 200
        data = resp.json()
        assert data["completion"] == "Hello from LLM"
        assert data["usage"]["inputTokens"] == 10
        assert data["usage"]["outputTokens"] == 5
        assert data["cacheHit"] is False
        assert "latencyMs" in data
        assert data["modelId"] == "llama3.2"

    def test_complete_cache_hit(self, client: TestClient) -> None:
        cached_response = {
            "completion": "Cached response",
            "usage": {"inputTokens": 5, "outputTokens": 3},
            "modelId": "llama3.2",
        }
        self._mock_cache.get = AsyncMock(return_value=cached_response)

        resp = client.post("/v1/llm/complete", json={
            "prompt": "Say hello",
            "feature": "test",
        })

        assert resp.status_code == 200
        data = resp.json()
        assert data["completion"] == "Cached response"
        assert data["cacheHit"] is True

    def test_complete_cache_bypass(self, client: TestClient) -> None:
        self._mock_ollama.generate = AsyncMock(return_value={
            "response": "Fresh response",
            "prompt_eval_count": 8,
            "eval_count": 4,
        })

        resp = client.post("/v1/llm/complete", json={
            "prompt": "Say hello",
            "feature": "test",
            "cache": False,
        })

        assert resp.status_code == 200
        data = resp.json()
        assert data["completion"] == "Fresh response"
        self._mock_cache.get.assert_not_awaited()

    def test_complete_ollama_unavailable(self, client: TestClient) -> None:
        from verso_llm_gateway.providers.ollama import OllamaUnavailableError

        self._mock_ollama.generate = AsyncMock(side_effect=OllamaUnavailableError())

        resp = client.post("/v1/llm/complete", json={
            "prompt": "Say hello",
            "feature": "test",
        })

        assert resp.status_code == 503
        assert "not available" in resp.json()["detail"]

    def test_embed_success(self, client: TestClient) -> None:
        self._mock_ollama.embed = AsyncMock(return_value={
            "embeddings": [[0.1, 0.2, 0.3, 0.4, 0.5]],
        })

        resp = client.post("/v1/llm/embed", json={
            "text": "Hello world",
        })

        assert resp.status_code == 200
        data = resp.json()
        assert data["embedding"] == [0.1, 0.2, 0.3, 0.4, 0.5]
        assert data["dimensions"] == 5
        assert data["modelId"] == "nomic-embed-text"

    def test_moderate_safe(self, client: TestClient) -> None:
        self._mock_ollama.generate = AsyncMock(return_value={
            "response": '{"verdict": "safe", "confidence": 0.95, "reason": "No issues found"}',
        })

        resp = client.post("/v1/llm/moderate", json={
            "content": "This is a normal book review",
            "contentType": "text",
        })

        assert resp.status_code == 200
        data = resp.json()
        assert data["verdict"] == "safe"
        assert data["confidence"] == 0.95

    def test_moderate_unparseable_response(self, client: TestClient) -> None:
        self._mock_ollama.generate = AsyncMock(return_value={
            "response": "I cannot parse this as JSON",
        })

        resp = client.post("/v1/llm/moderate", json={
            "content": "Test content",
        })

        assert resp.status_code == 200
        data = resp.json()
        assert data["verdict"] == "review"
        assert data["confidence"] == 0.0

    def test_complete_with_model_tier(self, client: TestClient) -> None:
        self._mock_ollama.generate = AsyncMock(return_value={
            "response": "Response",
            "prompt_eval_count": 5,
            "eval_count": 3,
        })

        resp = client.post("/v1/llm/complete", json={
            "prompt": "Hello",
            "feature": "search",
            "modelTier": "default",
        })

        assert resp.status_code == 200
