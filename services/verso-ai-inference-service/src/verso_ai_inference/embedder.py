from __future__ import annotations

import httpx
import pybreaker
from verso.logger import create_logger

from verso_ai_inference.config import settings

logger = create_logger("verso-ai-inference.embedder")

# Circuit breaker for LLM gateway outbound calls — opens after 5 consecutive
# failures and resets after 30 s, preventing cascading timeouts.
gateway_breaker = pybreaker.CircuitBreaker(fail_max=5, reset_timeout=30, name="llm-gateway")


class EmbeddingResult:
    """Result from the LLM gateway embed endpoint."""

    __slots__ = ("embedding", "dimensions", "model_id")

    def __init__(self, embedding: list[float], dimensions: int, model_id: str) -> None:
        self.embedding = embedding
        self.dimensions = dimensions
        self.model_id = model_id


class Embedder:
    """HTTP client for the LLM gateway's /v1/llm/embed endpoint."""

    def __init__(self) -> None:
        self._client: httpx.AsyncClient | None = None

    def set_client(self, client: httpx.AsyncClient) -> None:
        self._client = client

    @property
    def client(self) -> httpx.AsyncClient:
        if self._client is None:
            msg = "Embedder HTTP client not initialized"
            raise RuntimeError(msg)
        return self._client

    async def embed(self, text: str) -> EmbeddingResult | None:
        """Call LLM gateway to generate an embedding. Returns None on failure."""
        try:
            return await self._do_embed(text)
        except pybreaker.CircuitBreakerError:
            logger.warning("llm_gateway_circuit_open", detail="Circuit breaker open — skipping embedding")
            return None

    @gateway_breaker
    async def _do_embed(self, text: str) -> EmbeddingResult | None:
        """Inner method wrapped by the circuit breaker."""
        url = f"{settings.llm_gateway_url}/v1/llm/embed"
        try:
            resp = await self.client.post(
                url,
                json={"text": text},
                timeout=settings.llm_gateway_timeout_seconds,
            )
            resp.raise_for_status()
            data = resp.json()
            return EmbeddingResult(
                embedding=data["embedding"],
                dimensions=data["dimensions"],
                model_id=data["modelId"],
            )
        except httpx.HTTPStatusError as exc:
            logger.warning(
                "llm_gateway_error",
                status=exc.response.status_code,
                detail=exc.response.text[:200],
            )
            raise  # Let breaker count this as a failure
        except (httpx.ConnectError, httpx.TimeoutException) as exc:
            logger.warning("llm_gateway_unavailable", error=str(exc))
            raise  # Let breaker count this as a failure

    async def close(self) -> None:
        if self._client:
            await self._client.aclose()


embedder = Embedder()
