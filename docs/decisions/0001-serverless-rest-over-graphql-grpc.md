# 0001 — Serverless REST replaces GraphQL + gRPC

Date: 2026-08-13 · Status: accepted

## Context

The original service was one long-running Go process exposing two surfaces
over one catalog: public read-only GraphQL (`:8080`) and trusted read/write
gRPC (`:50051`), backed by PostgreSQL. The mandate: move to AWS Lambda +
DynamoDB in a monorepo.

## Decision

Replace both surfaces with a single REST API on API Gateway (HTTP API),
split across three Lambdas by aggregate (games / sets / cards):

- **Reads are public, writes require a bearer token** — the same trust
  split GraphQL (public) vs gRPC (`API_TOKEN`) provided, now enforced in one
  place (`apps/api/internal/httpapi`).
- **gRPC is dropped**: API Gateway HTTP APIs can't speak native gRPC, and the
  only gRPC consumers (scrapers/importers) are HTTP-capable — the Python
  importer covers bulk loading.
- **GraphQL is dropped**: the schema was six flat queries with no nested
  resolution worth the runtime; REST routes map 1:1 to them
  (`searchCards(game, query)` → `GET /v1/games/{game}/cards?query=`).

## Consequences

- One protocol, one auth model, one place to document (docs/api.md).
- Old gRPC/GraphQL clients need a port — route-for-route mapping exists.
- Per-route Lambda metrics/alarms come free from the aggregate split.
- If a graph query layer is ever needed again, AppSync or a `graphql` Lambda
  can front the same `libs/go/catalog` store.
