# AGENTS.md — Verso platform

Operating rules for any agent/session working in this repo. This file is auto-loaded — treat it as
binding.

> **Start every session:** read `../docs/07-build-plan.md`, pick the next unit.
> **End every session:** tick that ledger + commit.
> **Specs `../docs/00`–`06` are LAW.** `00-vision-and-spec.md` is the canon.

---

## What this is
**Verso** — a social-reading + AI + digital-publishing platform, built as microservices that run on
**docker-compose**. The goal is to demonstrate architecture and domain modeling. It is not a
commercial product: **commercial features are modeled, not monetized** (payments are sandbox-stubbed).

## Golden rules (non-negotiable)
- **Specs are law.** Don't invent service/event/entity names not in `../docs/00`. If something is
  genuinely missing, add it to `00` first (and flag under "Proposed additions").
- **Event-first.** Default coupling = **events over Redpanda** (Kafka API). Sync calls only for true
  command dependencies (mostly the purchase saga).
- **Internal sync transport = REST/JSON** (Connect/buf optional). **No gRPC.**
- **Edge = Traefik → per-client BFF (Node/Fastify) → services.** Clients never call services directly.
- **Schema-per-service. No cross-service foreign keys.** Reference other services by **ULID** id.
- **IDs = ULID** (`CHAR(26)`), opaque strings in APIs. No sequential ints in public APIs.
- **docker-compose only.** No Kubernetes / service mesh / Terraform / cloud. Resilience is **in-app**.
- **No premature scaling.** Autoscaling / sharding / multi-region / CDN = *future notes* only, not built.

## Repo layout
```
platform/
├── AGENTS.md
├── services/<verso-x-service>/    # one dir per backend service
├── web/                           # Next.js + React + TS + Tailwind
├── mobile/                        # Flutter
├── proto/                         # Protobuf event (+ optional Connect) schemas — contracts-first
├── libs/{go,node,py}/             # shared: event envelope, outbox, otel, errors, logger, jwt
└── deploy/docker-compose.yml
```

## Stack (right tool per workload)
- **Go** — core domain services · `chi`, `pgx`+`sqlc`, `franz-go`, `sony/gobreaker`, `goose`, OTel
- **Node + Fastify (TS)** — realtime + BFF · `fastify`, `kafkajs`, `ws`, `opossum`, `drizzle`/`pg`, `zod`, OTel
- **Python + FastAPI** — AI/ML · `fastapi`, `asyncpg`+`pgvector`, `aiokafka`, `pybreaker`+`tenacity`, OTel
- **Infra:** Postgres+pgvector · Redis · Redpanda · OpenSearch · MinIO · Traefik · MailHog
  (profiles: `observability` = OTel/Grafana/Prometheus/Loki/Tempo · `ai-local` = Ollama)

## Which service uses which language / store
Authoritative table: `../docs/00-vision-and-spec.md` §9.3. Per-service detail: `../docs/06-service-tech-stack.md`.

## Naming & API conventions
- Service: `verso-<domain>-service` · Event topic: `verso.<domain>.<event>.v<major>` · REST: `/v1/...` (plural, kebab-case)
- Errors: **RFC 9457** (problem+json) · Pagination: **cursor** · Time: **ISO-8601 UTC**
- **Idempotency:** write APIs accept `Idempotency-Key`; event consumers dedupe on `eventId`
- **Auth:** JWT issued by `verso-identity-service`; validate via JWKS; forward token in `Authorization`
- **Events:** Protobuf in `proto/`, registered in Redpanda Schema Registry; envelope carries
  `eventId, type, occurredAt, producer, traceId, partitionKey, payload, schemaVersion`

## Service skeleton (every service has this)
health + readiness · env config · structured logging · OTel init · Dockerfile · migrations
(schema-per-service, ULID, no cross-svc FK) · REST per `06` · events per `00 §9.4` with **outbox** ·
Traefik labels · in-app resilience · unit + **testcontainers** tests.
→ Full **Definition of Done**: `../docs/07-build-plan.md`.

## Verify before claiming done
- `docker compose up -d` → all healthy
- service tests green (unit + testcontainers)
- lsp/build clean; no suppressed type/lint errors
- one smoke request through the BFF works

## Do NOT
- Use **gRPC** (as transport), **Kubernetes**, a service mesh, or any cloud dependency.
- Add **cross-service DB foreign keys** or read another service's schema directly.
- Rename or add canonical services/events/entities without first updating `../docs/00`.
- Suppress type/lint errors, leave `docker-compose` broken, or build many services in one session.

## Workflow (one unit per session)
`read ../docs/07-build-plan.md` → **plan mode** (load only the relevant spec slices) → finalize →
**`/start-work`** → build → **verify** (above) → tick the ledger → **commit**.

## Git identity
Name: Shahadul Haider · Email: shahadul.haider@gmail.com (global gitconfig already set — don't override).
