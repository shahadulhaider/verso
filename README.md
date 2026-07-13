# Verso Platform

Social reading platform. AI reading companion. Digital publishing marketplace.

> *"Verso"* -- the left-hand page of an open book. Where books find their readers.

## What is this?

Verso is a polyglot microservices platform built on docker-compose. It demonstrates event-driven architecture, CQRS, saga orchestration, and real service boundaries across Go, Node/TypeScript, and Python. Commercial features are modeled, not monetized (payments are sandbox-stubbed). The goal is architecture and domain modeling, not a production launch.

## Quick Start

```bash
# 1. Clone and setup
cp .env.example .env

# 2. Start infrastructure
task up

# 3. Verify all containers are healthy
task ps
```

## Architecture Overview

- **Polyglot:** Go (core domain), Node/Fastify (realtime + BFF), Python/FastAPI (AI/ML)
- **Event-first:** Redpanda (Kafka API) + Protobuf events + transactional outbox + Debezium CDC
- **Edge:** Traefik -> BFF -> services (clients never call services directly)
- **Storage:** Postgres+pgvector (schema-per-service), Redis, OpenSearch, MinIO
- **IDs:** ULID everywhere, opaque strings in APIs
- **Sync transport:** REST/JSON. No gRPC.

## Repository Layout

```
platform/
├── services/               # Backend microservices (Phase 1+)
├── web/                    # Next.js + React frontend (Phase 1+)
├── mobile/                 # Flutter mobile app (Phase 3+)
├── proto/                  # Protobuf event schemas (buf)
│   └── verso/common/v1/   # Event envelope, error types
├── libs/
│   ├── go/                 # Shared Go libs (envelope, outbox, otel, errors, logger, jwt)
│   ├── node/               # Shared Node/TS libs (@verso/* packages)
│   └── py/                 # Shared Python libs (verso.* modules)
├── gen/                    # Generated code from proto/ (buf generate)
│   ├── go/
│   ├── ts/
│   └── py/
├── deploy/
│   ├── docker-compose.yml  # All infrastructure + profiles
│   ├── config/             # OTel, Prometheus, Grafana, Tempo configs
│   └── init/               # Postgres init scripts
├── Taskfile.yml            # Task runner (task up, task down, etc.)
├── .env.example            # Environment variables template
└── AGENTS.md               # Agent conventions (auto-loaded)
```

## Infrastructure

| Container | Image | Port | Purpose |
|---|---|---|---|
| postgres | pgvector/pgvector:pg16 | 5432 | Primary DB + vector search |
| redis | redis:7-alpine | 6379 | Cache, rate limiting, job queues |
| redpanda | redpandadata/redpanda | 19092 | Event bus (Kafka API) + Schema Registry |
| redpanda-console | redpandadata/console | 8888 | Kafka UI |
| opensearch | opensearchproject/opensearch | 9200 | Full-text search |
| opensearch-dashboards | opensearchproject/opensearch-dashboards | 5601 | Search UI |
| minio | minio/minio | 9000/9001 | S3-compatible object storage |
| traefik | traefik:v3 | 80/443/8080 | API gateway + dashboard |
| mailhog | mailhog/mailhog | 1025/8025 | Dev SMTP sink |
| debezium | debezium/connect | 8083 | CDC for outbox pattern |

## Compose Profiles

```bash
# Core infrastructure only
task up

# Add observability (OTel, Prometheus, Grafana, Loki, Tempo)
docker compose -f deploy/docker-compose.yml --profile observability up -d

# Add local AI (Ollama)
docker compose -f deploy/docker-compose.yml --profile ai-local up -d
```

## Taskfile Targets

| Target | Description |
|---|---|
| `task up` | Start all infrastructure containers |
| `task down` | Stop containers (preserve data) |
| `task nuke` | Stop + delete all volumes (clean slate) |
| `task test` | Run all tests (Go + Node + Python) |
| `task gen` | Generate code from proto definitions |
| `task lint` | Lint all code |
| `task debezium:register` | Register CDC connector (after `task up`) |
| `task ps` | Show container status |
| `task logs` | Tail container logs |

## Shared Libraries

Each language has a parallel set of shared modules used by its services:

- **Go** (`libs/go/`): envelope, outbox, otel, errors, logger, jwt. Used by Go domain services.
- **Node** (`libs/node/`): @verso/envelope, @verso/outbox, @verso/otel, @verso/errors, @verso/logger, @verso/jwt. Used by BFFs and Node services.
- **Python** (`libs/py/`): verso.envelope, verso.outbox, verso.otel, verso.errors, verso.logger, verso.jwt. Used by AI/ML services.

## Development Prerequisites

- Docker Desktop (4GB+ RAM recommended)
- Go 1.22+
- Node.js 20+ with pnpm
- Python 3.11+
- [Task](https://taskfile.dev/) (task runner)
- [buf](https://buf.build/) (protobuf tooling)

## Tech Stack

| Layer | Technology |
|---|---|
| Go services | chi, pgx+sqlc, franz-go, gobreaker, goose |
| Node services | Fastify, kafkajs, opossum, drizzle, pino |
| Python services | FastAPI, asyncpg, aiokafka, pybreaker, structlog |
| Web frontend | Next.js, React, TypeScript, Tailwind |
| Mobile | Flutter |
| Events | Protobuf + Redpanda (Kafka API) + Debezium CDC |
| Gateway | Traefik v3 |

## Build Phases

- **Phase 0** (done) -- Foundation: infrastructure, shared libs, tooling
- **Phase 1** -- Walking skeleton: identity, catalog, BFF, search
- **Phase 2** -- MVP: profiles, media, library, reviews, social, feed, AI
- **Phase 3** -- Retention: recommendations, clubs, gamification, messaging
- **Phase 4** -- Marketplace: publishing, reader, commerce, payouts

See [`../docs/07-build-plan.md`](../docs/07-build-plan.md) for the full progress ledger.

## Conventions

Development conventions, naming rules, API patterns, and the Definition of Done live in [`AGENTS.md`](AGENTS.md). Full specifications are in [`../docs/`](../docs/).

## License

(c) 2025-2026 Shahadul Haider. All rights reserved.
