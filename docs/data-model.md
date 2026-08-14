# Data model (DynamoDB single table)

One table holds the whole catalog. The authoritative layout lives as code in
[libs/card-catalog-store/dynamo.go](../libs/card-catalog-store/dynamo.go); this page explains it.

## Domain rules (carried over from the PostgreSQL era)

- **Prices live on card variants**, overwritten in place — never on the card.
- **Games and sets have a GUID `id` plus an immutable catalog `key`**
  (`pokemon`, `sv3pt5`); lookups accept either. Cards have a GUID plus an
  optional `tcgplayerId`, and lookups accept either.
- **Languages are separate catalogs.** `language` is ISO 639 alpha-3
  (`eng`, `jpn`); a Japanese card is its own row, never a translation.
- **Set card counts** used to be `count(*)`; DynamoDB can't afford that per
  read, so `cardCount` is maintained transactionally with every card write
  (see [decision 0002](decisions/0002-dynamodb-single-table.md)).

## Item types

| Entity | PK | SK | Notes |
| --- | --- | --- | --- |
| game | `GAME#{id}` | `META` | `GSI1PK=GAME`, `GSI1SK={createdAt}#{id}` — lists all games in creation order |
| game key guard | `GAMEKEY#{key}` | `UNIQ` | `refId` → game GUID |
| set | `GAME#{gameId}` | `SET#{id}` | a game's sets are one `Query` on the game partition |
| set key guard | `SETKEY#{gameId}/{key}` | `UNIQ` | `refId` → set GUID |
| card | `SET#{setId}` | `CARD#{id}` | a set's cards are one `Query`; variants embedded as a list attribute |

Card GSI attributes:

- `GSI1PK=GAMECARDS#{gameId}`, `GSI1SK={nameLower}#{id}` — per-game name
  search (`contains` filter on `nameLower`, capped at 48 results).
- `GSI2PK=CARD#{id}` — direct lookup by GUID (the set partition isn't known
  from the URL).
- `GSI3PK=TCGP#{gameId}#{tcgplayerId}` — sparse; lookup by TCGplayer id.

## Uniqueness via guard items

DynamoDB can't enforce uniqueness on a GSI, so every catalog key is reserved
by a **guard item** written in the same `TransactWriteItems` as the entity,
conditioned on `attribute_not_exists(PK)`. A duplicate key cancels the whole
transaction → the API returns 409. The guard doubles as the key→GUID lookup
(`refId`), so resolving `pokemon` or `sv3pt5` is a `GetItem`, not a scan.
Deleting an entity deletes its guard, freeing the key.

## Access patterns → operations

| API need | DynamoDB operation |
| --- | --- |
| list games | `Query GSI1, GSI1PK = GAME` |
| resolve game/set by key | `GetItem` guard → `GetItem` entity |
| list a game's sets | `Query PK = GAME#{gameId}, SK begins_with SET#` (name filter + release-date sort in code) |
| list a set's cards | `Query PK = SET#{setId}, SK begins_with CARD#` (collector-number sort in code) |
| search cards by name | `Query GSI1, GSI1PK = GAMECARDS#{gameId}, filter contains(nameLower, q)` |
| get card by GUID / TCGplayer id | `Query GSI2` / `Query GSI3` |
| create game/set | `TransactWriteItems`: put guard (condition) + put entity |
| create/delete card | `TransactWriteItems`: put/delete card + `ADD cardCount ±1` on its set |
| move card between sets | `TransactWriteItems`: delete + put + two counter updates |
| delete set | `BatchWriteItem` the cards (25/batch), then delete set + guard |

## Known trade-offs

- **Name search is a filtered Query**, not a text index — fine at catalog
  scale (tens of thousands of cards per game); if it outgrows that, front it
  with OpenSearch or Athena rather than bending the table.
- **Search ordering** is alphabetical (was: set release date). The GSI sort
  key gets results roughly ordered; code finishes the sort.
- **`updatedAt` is set by application code** (the old schema used a trigger);
  DynamoDB has no server-side equivalent.
