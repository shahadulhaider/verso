# 07 — Build Plan & Progress Ledger

> **This is the progress source-of-truth.** Design lives in `00`–`06` (stable). *What's done / what's
> next* lives here. This is a **living document** — update it every session.
> Status: Living · Conforms to `00`–`06`.

---

## How to use this ledger (per session)

1. **Read this file** → pick the next unbuilt unit (respect phase order & dependencies).
2. **Plan mode** → scope *only that unit*. Feed it the spec slices named in the unit's row (not the whole doc set).
3. **Finalize the plan** → run **`/start-work`**.
4. **Verify** against the **Definition of Done** below.
5. **Tick the boxes**, add a dated note, **commit**.

**One unit per session.** Don't build many services at once. Detailed plans are generated
just-in-time in plan mode; this ledger stays high-level and stable.

**Legend:** `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked

---

## Definition of Done (applies to every service)

- [ ] Skeleton: health + readiness endpoints, 12-factor env config, structured logging, OTel init
- [ ] Dockerfile (multi-stage, slim) **and** added to `deploy/docker-compose.yml`
- [ ] DB migrations — **schema-per-service, ULID ids, no cross-service FK** (per `05`)
- [ ] REST endpoints per the service's `06` row + `03` (documented via OpenAPI)
- [ ] Events produced/consumed per `00 §9.4` — **Protobuf schema in `proto/`**, registered
- [ ] **Outbox** pattern for every state-change event
- [ ] Traefik router labels; reachable through gateway → BFF
- [ ] In-app resilience (timeout, retry+backoff, circuit breaker) on outbound calls
- [ ] Tests: unit + **testcontainers** (real Postgres/Redpanda) — green
- [ ] `docker compose up` still green end-to-end
- [ ] This ledger updated + committed

---

## Phase 0 — Foundation  *(goal: `docker compose up` = whole infra online, green)*

- [x] Monorepo skeleton: `services/  web/  mobile/  proto/  libs/{go,node,py}/  deploy/`
- [x] `deploy/docker-compose.yml`: **Postgres+pgvector · Redis · Redpanda(+console) · OpenSearch(+dashboards) · MinIO(+console) · Traefik · MailHog** — all with healthchecks
- [x] `.env.example` + per-service env convention
- [x] Compose profiles: `observability` (OTel Collector, Prometheus, Grafana, Loki, Tempo) · `ai-local` (Ollama)
- [x] Shared libs (per language): event envelope · outbox helper · OTel bootstrap · RFC-9457 errors · logger · JWT/JWKS middleware
- [x] `proto/` + **buf** config + codegen pipeline (events; optional Connect stubs)
- [x] Makefile/Taskfile: `up down migrate test gen lint`
- [x] `platform/AGENTS.md` conventions (auto-loaded) — created

> **2026-07-13:** Phase 0 complete. All infra containers healthy (pg17, Redis 8.8, Redpanda v26,
> OpenSearch 2.17, Debezium 3.0, Traefik v3.7). Shared libs (Go/Node/Python) with tests. Taskfile,
> proto/buf pipeline, observability + ai-local profiles. Repo: github.com/shahadulhaider/verso

## Phase 1 — Walking skeleton  *(prove every layer once)*

- [x] `verso-identity-service` (Go) — register/login, JWT issue, JWKS · specs: `00 §9.3`, `05` identity, `06`
- [x] `verso-catalog-service` (Go) — Works/Editions/Formats CRUD, outbox → Redpanda · `00 §9.1/§9.2`, `05` catalog, `06`, `03`
- [x] `web-bff` (Node/Fastify) + minimal **Next.js** book-list page · `06` BFF, `03` edge
- [x] Search-indexer slice: catalog `work-created` → consumer → OpenSearch → search endpoint · `00 §9.4`, `03`
- [x] ✅ **Milestone:** client → Traefik → BFF → service → Postgres **+** outbox → Redpanda → consumer → OpenSearch

> **2026-07-14:** Phase 1 complete. Identity service (Go), catalog service (Go), search service
> (Go), web BFF (Node/Fastify), Next.js web app. Full pipeline: Traefik → BFF → services →
> Postgres → outbox → Debezium → Redpanda → search indexer → OpenSearch. All unit +
> integration tests pass. PR: phase-1/walking-skeleton

## Phase 2 — MVP / P0  *(per PRD `02`)*

- [x] `verso-profile-service` (Go)
- [x] `verso-media-service` (Node) — cover upload/serve via MinIO
- [x] `verso-library-service` (Go) — shelves, reading sessions, progress
- [x] `verso-review-service` (Go) — ratings + reviews + aggregate
- [x] `verso-social-service` (Go) — follow graph
- [x] `verso-feed-service` (Node) — fan-out consumer → Redis timeline (**first CQRS read model**)
- [x] `verso-llm-gateway` (Python) — model routing, cache, guardrails (Ollama in dev)
- [x] `verso-ai-inference-service` (Python) — embeddings → pgvector
- [x] `verso-search-service` (Go) — hybrid full-text + **semantic "books like X"** (the P0 AI feature)
- [x] Seed catalog from an **Open Library** dump
- [x] ✅ **Milestone:** MVP loop — discover · shelve · rate/review · follow · feed · semantic search

> **2026-07-16:** Phase 2 complete. Profile service (Go), media service (Node/Fastify), library service (Go), review service (Go), social service (Go), feed service (Node/Fastify), LLM gateway (Python/FastAPI), AI inference service (Python/FastAPI). Search upgraded with semantic vector search via pgvector. 50+ works seeded. Full MVP UI in Next.js. PR: phase-2/mvp

## Phase 3 — P1 Retention

- [ ] `verso-recommendation-service` (Python)
- [ ] **RAG "Ask this book"** (ai-inference + reader content access, spoiler-gated)
- [ ] `verso-club-service` (Go) · `verso-list-service` (Go) · `verso-gamification-service` (Go)
- [ ] `verso-notification-service` (Node) — full push/email/in-app
- [ ] `verso-messaging-service` (Node) — DMs/club chat (WebSockets)
- [ ] `verso-trust-safety-service` (Python) — **basic** moderation queue
- [ ] `mobile-bff` (Node) + **Flutter** app
- [ ] Annotations & highlights (reader-lite)

## Phase 4 — P2 Marketplace

- [ ] `verso-publishing-service` (Go) — ingestion pipeline
- [ ] `verso-reader-service` (Go) — ebook delivery, entitlement check, position/highlight sync
- [ ] `verso-commerce-service` (Go) — products/orders/entitlements + **saga orchestrator** (sandbox payments)
- [ ] `verso-payout-service` (Go) — royalty ledger
- [ ] ARC program
- [ ] ✅ **Milestone:** purchase → entitlement → royalty → notification **saga** end-to-end

## P3 — Stretch (optional)

- [ ] `verso-trust-safety-service` — ML fake-review / brigading / fraud detection
- [ ] `verso-analytics-service` (Go) — rollups & publisher dashboards
- [ ] `verso-admin-service` (Go) — back-office & feature flags
- [ ] Reading-pass subscription flows; full observability profile polish

---

## Service status board

| # | Service | Lang | Phase | Status | Session note |
|---|---|---|---|---|---|
| 1 | `verso-identity-service` | Go | P1 | [x] | Phase 1 — JWT/JWKS, auth endpoints, outbox |
| 2 | `verso-catalog-service` | Go | P1 | [x] | Phase 1 — Work/Edition/Format CRUD, outbox |
| 3 | `web-bff` | Node | P1 | [x] | Phase 1 — aggregation proxy to backend services |
| 4 | `web` (Next.js) | TS | P1→P2 | [x] | Phase 2 — full MVP UI (feed, library, profile, review, discover, social) |
| 5 | `verso-profile-service` | Go | P2 | [x] | Phase 2 — profiles, event-driven creation |
| 6 | `verso-media-service` | Node | P2 | [x] | Phase 2 — MinIO upload/serve |
| 7 | `verso-library-service` | Go | P2 | [x] | Phase 2 — shelves, reading progress |
| 8 | `verso-review-service` | Go | P2 | [x] | Phase 2 — ratings, reviews, votes, comments |
| 9 | `verso-social-service` | Go | P2 | [x] | Phase 2 — follow graph, blocks |
| 10 | `verso-feed-service` | Node | P2 | [x] | Phase 2 — activity fan-out, Redis timelines |
| 11 | `verso-llm-gateway` | Python | P2 | [x] | Phase 2 — Ollama routing, Redis cache |
| 12 | `verso-ai-inference-service` | Python | P2 | [x] | Phase 2 — embeddings → pgvector |
| 13 | `verso-search-service` | Go | P1→P2 | [x] | Phase 2 — semantic + hybrid search |
| 14 | `verso-recommendation-service` | Python | P3 | [ ] | |
| 15 | `verso-club-service` | Go | P3 | [ ] | |
| 16 | `verso-list-service` | Go | P3 | [ ] | |
| 17 | `verso-gamification-service` | Go | P3 | [ ] | |
| 18 | `verso-notification-service` | Node | P3 | [ ] | |
| 19 | `verso-messaging-service` | Node | P3 | [ ] | |
| 20 | `verso-trust-safety-service` | Python | P3 | [ ] | basic; ML in stretch |
| 21 | `mobile-bff` | Node | P3 | [ ] | |
| 22 | `mobile` (Flutter) | Dart | P3→ | [ ] | |
| 23 | `verso-publishing-service` | Go | P4 | [ ] | |
| 24 | `verso-reader-service` | Go | P4 | [ ] | |
| 25 | `verso-commerce-service` | Go | P4 | [ ] | saga orchestrator |
| 26 | `verso-payout-service` | Go | P4 | [ ] | |
| 27 | `verso-analytics-service` | Go | Stretch | [ ] | |
| 28 | `verso-admin-service` | Go | Stretch | [ ] | |

*(24 backend services + web, web-bff, mobile, mobile-bff.)*

---

## Per-service checklist template (copy per service into a session's plan)

```
### <verso-x-service>  (Phase _, Lang _)
Spec slices to load: 00 §9.3 row, 05 <context> schema, 06 <service> row, 03 <service> section
- [ ] scaffold from service template (health, config, OTel, Dockerfile)
- [ ] migrations (schema `<x>`, ULID, no cross-svc FK)
- [ ] REST endpoints: <list from 06/03>
- [ ] events produced: <from 00 §9.4>
- [ ] events consumed: <from 00 §9.4>  (idempotent handlers)
- [ ] outbox wired for produced events
- [ ] Traefik labels + BFF wiring
- [ ] resilience on outbound calls
- [ ] tests (unit + testcontainers) green
- [ ] added to docker-compose; `compose up` green
- [ ] ledger ticked + commit
```

## Cross-cutting foundations (build once, reuse everywhere)
- [ ] Event envelope + outbox library (per language)
- [ ] OTel bootstrap + trace propagation (HTTP headers + event envelope)
- [ ] Error format (RFC 9457) + logger
- [ ] JWT/JWKS auth middleware
- [ ] `proto/` event contracts + buf codegen
- [ ] Testcontainers harness + one smoke E2E through the BFF
- [ ] CI: lint + test + build images (docker-compose based)
