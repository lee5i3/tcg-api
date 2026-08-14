# REST API

Base URL: `terraform output api_endpoint` (API Gateway HTTP API).
All bodies are JSON with camelCase fields. Reads are public; **writes
(POST/PUT/DELETE) require `Authorization: Bearer <API_TOKEN>`** (401
otherwise). When the deployment sets no token, writes are open — dev only.

Path references are flexible: `{game}` and `{set}` accept the GUID **or** the
immutable catalog key (`pokemon`, `sv3pt5`); `{card}` accepts the GUID **or**
the numeric TCGplayer product id.

## Errors

```json
{ "error": "not found: game \"pokmon\"" }
```

| Status | Meaning |
| --- | --- |
| 400 | invalid input (bad key/language/date, unknown set on card create, malformed JSON) |
| 401 | missing/invalid bearer token on a write route |
| 404 | game/set/card/route not found |
| 409 | key already exists (game/set create) |
| 500 | unexpected — details only in CloudWatch, never in the response |

## Accounts & auth — λ `api-auth-routes`

End-user accounts for the app (JWT sessions, HS256 via `USER_JWT_SECRET`) —
distinct from the admin's shared `API_TOKEN`:

| Route | Auth | Response |
| --- | --- | --- |
| `POST /v1/auth/register` | public | 201 `{"token","user"}`; 409 email taken; 400 bad email / password < 8 chars |
| `POST /v1/auth/login` | public | 200 `{"token","user"}`; 401 `{"error":"invalid email or password"}` |
| `GET /v1/auth/me` | user JWT | 200 `{"user"}`; 401 missing/invalid/expired |
| `POST /v1/auth/check` | admin token | 204 when the bearer token is valid (the admin site's login) |
| `GET /v1/auth/providers` | public | `{"providers":[{"id","label"}]}` — social providers with configured credentials |
| `GET /v1/auth/oauth/{provider}/start?redirect=` | public | 302 to the provider's consent screen (signed CSRF state, 10-min TTL) |
| `GET`/`POST /v1/auth/oauth/{provider}/callback` | public | 302 to the app: `/auth/callback#token=…&redirect=…` on success, `/login?error=…` on failure (POST variant is Apple's form_post) |

**Social sign-in** (Google, Facebook, Apple): standard authorization-code
flow, exchanged server-side. Accounts are matched by verified email — signing
in with Google using the email of an existing password account links the
identity to that account; brand-new identities create social-only accounts
(no password). A provider activates only when its `*_CLIENT_ID` +
`*_CLIENT_SECRET` are configured on the auth Lambda, plus `APP_URL` (the
redirect_uri host). Register `{APP_URL}/v1/auth/oauth/{provider}/callback` in
each provider's console.

Register body: `{"email","password","name"?}`; login body `{"email","password"}`.
User: `{"id","email","name","createdAt"}`. Send the session as
`Authorization: Bearer <token>`; tokens last 7 days.

The **app requires sign-in** (client-side gate + these endpoints); the
catalog read endpoints below remain public API surface for services.

## Games & service — λ `api-game-routes`

| Route | Auth | Response |
| --- | --- | --- |
| `GET /healthz` | — | `{"status":"ok"}` |
| `GET /v1/games` | — | `{"games":[Game]}` |
| `POST /v1/games` | ✅ | 201 `{"game":Game}` |

`POST /v1/games` body: `{"key":"lorcana","label":"Disney Lorcana","language":"eng"?}`
— key is 2–32 chars of `[a-z0-9-]`, immutable. Language is ISO 639 alpha-3,
default `eng`, immutable.

Game: `{"id","key","language","label","updatedAt"}`

## Sets — λ `api-set-routes`

| Route | Auth | Response |
| --- | --- | --- |
| `GET /v1/games/{game}/sets?query=` | — | `{"sets":[SetSummary]}` newest release first; `query` filters by name substring |
| `POST /v1/games/{game}/sets` | ✅ | 201 `{"id":"<guid>"}` |
| `PUT /v1/games/{game}/sets/{set}` | ✅ | 204 |
| `DELETE /v1/games/{game}/sets/{set}` | ✅ | `{"cardsDeleted":n}` — cards cascade |

Set input body: `{"key","name","language"?,"releaseDate"?,"cardTotal"?,"logoUrl"?}`
— `releaseDate` is `YYYY-MM-DD` (`YYYY/MM/DD` accepted). On update, `key` and
`language` are ignored (immutable); omitted optional fields are cleared.

SetSummary: `{"id","key","language","gameId","name","cardCount","releaseDate","cardTotal","logoUrl","createdAt","updatedAt"}`

## Cards — λ `api-card-routes`

| Route | Auth | Response |
| --- | --- | --- |
| `GET /v1/games/{game}/sets/{set}/cards` | — | `{"cards":[CardSummary]}` collector-number order |
| `GET /v1/games/{game}/cards?query=` | — | `{"cards":[CardSummary]}` name search, ≤48 results, alphabetical; empty query → `[]` |
| `GET /v1/games/{game}/cards/{card}` | — | `{"card":CardDetail}` |
| `POST /v1/games/{game}/cards` | ✅ | 201 `{"id":"<guid>"}` |
| `PUT /v1/games/{game}/cards/{card}` | ✅ | 204 |
| `DELETE /v1/games/{game}/cards/{card}` | ✅ | 204 |

Card input body:
`{"setId","name","number"?,"rarity"?,"language"?,"tcgplayerId"?,"imageSmall"?,"imageLarge"?}`
— `setId` accepts set GUID or key. On update, `language` is ignored
(immutable) and variants are preserved.

CardSummary: `{"id","tcgplayerId","language","name","number","rarity","setId","image","imageLarge","variants":[{"id","name","price"}],"createdAt","updatedAt"}`

CardDetail: same, plus `"gameId"`, `"set"` (set GUID), and
`"images":{"small","large"}` instead of the flat image fields.

## Examples

```sh
API=$(cd infra/terraform && terraform output -raw api_endpoint)

curl -s "$API/v1/games"
curl -s "$API/v1/games/pokemon/sets?query=151"
curl -s "$API/v1/games/pokemon/cards?query=zapdos"

curl -s -X POST "$API/v1/games/pokemon/sets" \
  -H "Authorization: Bearer $TCG_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key":"sv3pt5","name":"151","releaseDate":"2023-09-22","cardTotal":165}'
```

Bulk loading: use the admin site (games/sets/cards CRUD) or script the write
endpoints above with the bearer token.

## Prices — λ `pokemon-price-updater` (no HTTP surface)

Variant prices are written by the scheduled `pokemon-price-updater` job, not
through the REST API. It pulls quotes from the feed configured as
`PRICE_API_URL` (`GET ?ids=501773,…` →
`{"prices":{"501773":{"Normal":1.42,"Holofoil":5.00}}}`) and overwrites each
card's variant prices in place — the catalog keeps no price history.
