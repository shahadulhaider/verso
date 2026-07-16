FROM python:3.12-slim AS builder
WORKDIR /build

# Copy shared Python libs (resolves verso-libs @ file:../../libs/py)
COPY libs/py/ libs/py/

# Copy service source + config
COPY services/verso-ai-inference-service/pyproject.toml services/verso-ai-inference-service/
COPY services/verso-ai-inference-service/src/ services/verso-ai-inference-service/src/

WORKDIR /build/services/verso-ai-inference-service
RUN pip install --no-cache-dir --prefix=/install .

FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

WORKDIR /app
COPY --from=builder /build/services/verso-ai-inference-service/src/ src/

ENV PYTHONUNBUFFERED=1

EXPOSE 8012

CMD ["uvicorn", "verso_ai_inference.main:app", "--host", "0.0.0.0", "--port", "8012"]
