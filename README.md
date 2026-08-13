# TCG API

A standalone trading-card catalog service in Go, backed by PostgreSQL. It
stores card sets, cards, and price history for one or more TCGs and exposes
two surfaces over one catalog:

- **gRPC** (`:50051`) — the trusted service-to-service API with full
  read/write access. Companion services (scrapers, pricing jobs, importers)
  push data in through it; the catalog never pulls from an upstream source.
- **GraphQL** (`:8080/graphql`) — the public read-only query API, with an
  embedded GraphiQL playground at [`/graphiql`](http://localhost:8080/graphiql).

## Data model

- **Prices are history, not fields.** Every price observation lands as a
  snapshot row in `card_prices`; a card holds `price_id`, a reference to its
  latest snapshot. Recording an unchanged price is a no-op
  (`recorded: false`), so pushers can submit unconditionally without
  flooding the history.
- **Set sizes are never stored.** `cardCount` is always a live count of the
  cards table.
- **Everything has a GUID `id` (first column) plus an immutable `key`.**
  Games (`pokemon`), sets (`sv3pt5`), and cards (`base1-4`) all follow the
  same pattern, and every lookup accepts either the GUID or the key.
- **Sets reference games and series by GUID.** A `series` table groups a
  game's sets ("Scarlet & Violet", "Sword & Shield"); series rows are created
  on demand when a set names one. Sets also carry `symbol` (what the set
  symbol depicts — "mew" for 151) and a real `date` release date.

Tables: `games`, `series`, `sets`, `cards`, `card_prices` — see
[internal/postgres/schema.sql](internal/postgres/schema.sql), applied
idempotently at boot.

## Running

```sh
make dev    # PostgreSQL 17 + the service, rebuilt on code changes
```

`make dev` runs `docker compose up --build --watch`. Other targets:
`make up` (detached), `make down`, `make logs`, `make check`
(build + vet + test + gofmt), and `make proto` (regenerate gRPC code).
Run `make` to list them all.

To skip Docker for the service itself:

```sh
docker compose up -d postgres
cp .env.example .env           # defaults work for local dev
go run ./cmd/api               # GraphQL on :8080, gRPC on :50051
```

## gRPC (service-to-service)

The contract lives in [proto/tcgapi/v1/catalog.proto](proto/tcgapi/v1/catalog.proto):
`CatalogService` covers games (list/create), sets (list/create/update/delete),
cards (list-by-set/search/get/create/update/delete), and prices
(record/history). Server reflection is enabled, so `grpcurl` works out of the
box:

```sh
grpcurl -plaintext localhost:50051 list tcgapi.v1.CatalogService

grpcurl -plaintext -d '{"game":"pokemon","query":"zapdos"}' \
  localhost:50051 tcgapi.v1.CatalogService/SearchCards

grpcurl -plaintext -d '{"game":"pokemon","card_id":"sv3pt5-145","price":1.42}' \
  localhost:50051 tcgapi.v1.CatalogService/RecordPrice
```

Auth: set `API_TOKEN` and every call (except reflection) must carry
`authorization: Bearer <token>` metadata. Leave it empty for local dev.

Regenerate after editing the proto with `make proto` (needs `buf`,
`protoc-gen-go`, and `protoc-gen-go-grpc` on your PATH — all installable
with `go install`).

## GraphQL (public)

Read-only queries at `POST /graphql`; schema in
[internal/graphql/schema.go](internal/graphql/schema.go). No mutations — all
writes go through gRPC. The endpoint is deliberately unauthenticated.

```sh
curl -s localhost:8080/graphql -H 'Content-Type: application/json' -d '{
  "query": "{ searchCards(game: \"pokemon\", query: \"zapdos\") { key name number price } }"
}'
```

Queries: `games`, `sets(game, query)`, `setCards(game, setId)`,
`searchCards(game, query)`, `card(game, id)`, `priceHistory(game, cardId)`.

The GraphiQL playground at `/graphiql` is fully embedded (no CDN assets),
consistent with the service being self-contained.
