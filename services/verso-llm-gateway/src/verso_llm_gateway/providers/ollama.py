from __future__ import annotations

import httpx

from verso_llm_gateway.config import settings


class OllamaError(Exception):
    def __init__(self, status_code: int, detail: str) -> None:
        self.status_code = status_code
        self.detail = detail
        super().__init__(detail)


class OllamaUnavailableError(OllamaError):
    def __init__(self) -> None:
        super().__init__(503, "Ollama is not available. Start with --profile ai-local.")


class OllamaClient:
    def __init__(self, client: httpx.AsyncClient | None = None) -> None:
        self._client = client

    @property
    def client(self) -> httpx.AsyncClient:
        if self._client is None:
            raise RuntimeError("OllamaClient not initialized — call set_client first")
        return self._client

    def set_client(self, client: httpx.AsyncClient) -> None:
        self._client = client

    async def generate(
        self,
        model: str,
        prompt: str,
        *,
        max_tokens: int | None = None,
        temperature: float | None = None,
    ) -> dict:
        body: dict = {"model": model, "prompt": prompt, "stream": False}
        if max_tokens is not None:
            body.setdefault("options", {})["num_predict"] = max_tokens
        if temperature is not None:
            body.setdefault("options", {})["temperature"] = temperature

        return await self._post("/api/generate", body)

    async def embed(self, model: str, text: str) -> dict:
        body = {"model": model, "input": text}
        return await self._post("/api/embed", body)

    async def is_healthy(self) -> bool:
        try:
            resp = await self.client.get(f"{settings.ollama_url}/api/tags")
            return resp.status_code == 200
        except (httpx.ConnectError, httpx.TimeoutException):
            return False

    async def _post(self, path: str, body: dict) -> dict:
        try:
            resp = await self.client.post(
                f"{settings.ollama_url}{path}",
                json=body,
                timeout=settings.ollama_timeout_seconds,
            )
        except httpx.ConnectError:
            raise OllamaUnavailableError()
        except httpx.TimeoutException:
            raise OllamaError(504, "Ollama request timed out")

        if resp.status_code == 404:
            raise OllamaError(404, f"Model not found: {body.get('model', 'unknown')}")
        if resp.status_code >= 400:
            raise OllamaError(resp.status_code, resp.text)

        return resp.json()


ollama_client = OllamaClient()
