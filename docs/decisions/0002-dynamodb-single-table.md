# 0002 — DynamoDB single-table design

Date: 2026-08-13 · Status: accepted

## Context

PostgreSQL held four relational tables (games, sets, cards, card_variants)
with FK cascades, `ILIKE` search, live `count(*)` set sizes, and DB-enforced
unique keys. DynamoDB has none of those primitives natively.

## Decision

One table, three GSIs, with each relational feature translated explicitly:

| PostgreSQL feature | DynamoDB translation |
| --- | --- |
| unique catalog keys | **guard items** (`GAMEKEY#…`, `SETKEY#…`) written transactionally with `attribute_not_exists`; guard doubles as key→GUID lookup |
| FK cascade set→cards | explicit `BatchWriteItem` delete in `DeleteSet` |
| live `count(*)` card counts | **`cardCount` counter on the set item**, bumped via `ADD` in the same `TransactWriteItems` as every card create/delete/move |
| `ILIKE '%q%'` search | `nameLower` attribute + `contains()` filter on a per-game GSI partition, capped at 48 |
| `card_variants` table | **embedded list on the card item** — variants were only ever read as a JSON aggregate and have no independent write path |
| `updated_at` trigger | set by store code |

Full item layout: [docs/data-model.md](../data-model.md) and
`libs/go/catalog/dynamo.go`.

## Alternatives considered

- **Live counts via `Select: COUNT` queries** — faithful to the old "never
  store set sizes" rule, but N+1 queries per set listing; rejected.
  The counter is transactional with every card write, so it can't drift
  under normal operation.
- **Table per entity** — simpler mental model, but key resolution and the
  card move/count transaction would span tables (transactions can span
  tables, but the single table keeps capacity, backups, and IAM to one
  resource; standard practice for this size).
- **OpenSearch for name search** — right answer at large scale, unjustified
  cost/ops today; the filter approach is contained in one method
  (`SearchCards`) and can be swapped later.

## Consequences

- Search ordering changed from set-release-date to alphabetical.
- Every card write is a transaction (2× WCU) — negligible at catalog volume.
- No ad-hoc SQL; new access patterns need index thought first (write them
  down in data-model.md).
