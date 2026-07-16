FROM python:3.12-slim AS builder

WORKDIR /build

COPY pyproject.toml .
RUN pip install --no-cache-dir --prefix=/install .

COPY src/ src/
RUN pip install --no-cache-dir --prefix=/install --no-deps .

FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

WORKDIR /app
COPY src/ src/

ENV PYTHONUNBUFFERED=1

EXPOSE 8011

CMD ["uvicorn", "verso_llm_gateway.main:app", "--host", "0.0.0.0", "--port", "8011"]
