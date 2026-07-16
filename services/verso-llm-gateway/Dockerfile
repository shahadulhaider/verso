FROM python:3.12-slim AS builder
WORKDIR /build

# Copy shared Python libs (resolves verso-libs @ file:../../libs/py)
COPY libs/py/ libs/py/

# Copy service source + config (src/ must exist for setuptools egg_info)
COPY services/verso-llm-gateway/pyproject.toml services/verso-llm-gateway/
COPY services/verso-llm-gateway/src/ services/verso-llm-gateway/src/

WORKDIR /build/services/verso-llm-gateway
RUN pip install --no-cache-dir --prefix=/install .

FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

WORKDIR /app
COPY --from=builder /build/services/verso-llm-gateway/src/ src/

ENV PYTHONUNBUFFERED=1

EXPOSE 8011

CMD ["uvicorn", "verso_llm_gateway.main:app", "--host", "0.0.0.0", "--port", "8011"]
