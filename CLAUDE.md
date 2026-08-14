# TCG Catalog Monorepo

Serverless trading-card catalog (games → sets → cards → variants) on AWS:
container-image Go Lambdas behind API Gateway, DynamoDB single-table storage,
three static SvelteKit sites (marketing, app, admin), Terraform in four
independently-appliable stacks — all driven by NX. **Two languages only:
TypeScript/Node for web, Go for Lambdas and tooling. No Python.**

> **Keep this file current.** Whenever requirements change — a new
> directive from the owner, a renamed project, a changed rule — update
> CLAUDE.md in the same session, along with a row in MEMORY.md and, for
> structural decisions, an ADR in docs/decisions/.

## Map

| Path (NX project) | What | Language |
| --- | --- | --- |
| `libs/card-catalog-store/` (`card-catalog-store`) | Card-catalog domain + DynamoDB persistence. **The single-table layout is documented in `dynamo.go` — read it before touching storage.** (Go package name is `catalog`.) | Go |
| `libs/httpapi/` (`httpapi`) | Shared Lambda plumbing: routing, bearer auth, error→status | Go |
| `libs/api-client/` (`api-client`) | `@tcg/api-client` — API types + fetch client for all sites | TypeScript |
| `libs/user-accounts/` (`user-accounts`) | End-user accounts: register/authenticate (bcrypt) in the single table | Go |
| `libs/design-system/` (`design-system`) | `@tcg/design-system` — shared tokens, base styles, and Svelte components; ALL sites share one look & feel | TypeScript/Svelte |
| `apps/api/{game,set,card,auth}-routes/` (`api-*-routes`) | One HTTP Lambda per aggregate — own NX project, Go module, **Dockerfile**. auth-routes owns `/v1/auth/*`: user register/login/me (JWT, `USER_JWT_SECRET`) + admin check | Go |
| `apps/jobs/pokemon-price-updater/` (`pokemon-price-updater`) | Scheduled job: variant prices from a JSON feed | Go |
| `apps/web/www/` (`www`) | Product front door: features, pricing, screenshots, CTA → app. Signup/checkout not built (marked TODO) | SvelteKit static |
| `apps/web/app/` (`app`) | Catalog browser — **starts at a login page** (register offered); user JWT in localStorage, catalog gated client-side | SvelteKit static |
| `apps/web/admin/` (`admin`) | Catalog management — token login (`POST /v1/auth/check`), separate site | SvelteKit static |
| `tools/local-gateway/` (`local-gateway`) | docker-compose stand-in for API Gateway (HTTP → Lambda RIE) | Go |
| `infra/terraform/stacks/{database,api,jobs,sites}` (`infra`) | Independently-appliable stacks, wired by remote state (database → api/jobs → sites). **Production only — no environment suffixes/variables** | HCL |
| `docs/` | architecture, data-model, api, development + `docs/decisions/` ADRs | |
| `MEMORY.md` | Log of Claude requests — append a row whenever a session materially changes the repo | |

**Sharing rules (ADR 0004–0006):** apps never import other apps' code —
shared code is promoted to a lib directly under `libs/`, **named specifically
for its function** (`card-catalog-store`, not `catalog`; never grouped by
language).

## Commands

```sh
npm install                  # once
npx nx run-many -t test      # all tests (Go + Vitest)
npx nx run-many -t lint      # go vet/gofmt, svelte-check/tsc, terraform fmt
docker compose up --build    # ENTIRE product locally: marketing :8082, app :8080,
                             # admin :8081, API :3000, DynamoDB :8000 (token: local-dev-token)
npx nx dev app               # frontend inner loop (VITE_API_URL=http://localhost:3000)
```

Sites build statically to `apps/<x>/dist`. Go patterns must be module-rooted
(`go test ./...` across modules is rejected — see docs/development.md).
Deploy: `infra/terraform/README.md` (per-stack; images via
`tools/scripts/push-images.sh` or the `build.yaml` workflow).

## CI (all PR-gating status checks)

`test` (test.yaml) · `checks-nodejs` (npm audit) · `checks-golang`
(govulncheck) · `build` (Docker builds, pushes to ECR on main) — mark all
four required in branch protection. `infra.yaml` = workflow_dispatch
terraform per stack. `dependabot.yaml` = weekly Actions/npm/gomod updates.

## Architecture rules

- **All storage goes through `libs/card-catalog-store`.** Handlers never
  touch DynamoDB; each Lambda declares a narrow `Store` interface — handler
  tests fake the store, store tests fake DynamoDB (`fake_dynamo_test.go`).
- **Domain errors are contracts**: wrap with `%w` around `catalog.ErrNotFound`
  / `ErrInvalid` / `ErrConflict`; `httpapi.Error` maps to 404/400/409;
  everything else is an opaque 500.
- **Two credential systems, don't mix them**: catalog writes use the admin
  `API_TOKEN` (httpapi.Route/Dispatch); app users get bcrypt accounts +
  7-day JWTs from auth-routes. User JWTs grant NO catalog write access.
  Catalog reads stay public API surface (ADR 0007) — the login wall is app UX.
- **Adding a route touches THREE places**: the Lambda's `Handler()` map,
  `infra/terraform/stacks/api/apigateway.tf` `locals.routes`, and
  `tools/local-gateway/main.go` `routeTable` — plus docs/api.md.
- **Changing the item layout**: update `libs/card-catalog-store/dynamo.go`'s
  header table, `docs/data-model.md`, `infra/terraform/stacks/database/main.tf`,
  and `tools/scripts/create-local-table.sh` together.
- Catalog invariants: keys and `language` immutable after create; variant
  prices overwritten in place (no history); `cardCount` maintained
  transactionally with every card write.
- **Web**: API access only through `@tcg/api-client`; UI styling/components
  only through `@tcg/design-system` (one look & feel across marketing/app/
  admin — the admin is identified by a badge, not a different theme); sites
  are static — never add SSR/server endpoints without an ADR.

## Gotchas

- History: PostgreSQL/GraphQL/gRPC → zip Lambdas → React sites → current
  shape; see `docs/decisions/0001…0008` before reintroducing anything.
- Terraform deploys straight to production (owner's call, 2026-08-14) —
  deletion protection and ECR retention default on; there are no dev/staging
  stacks. Fresh AWS account: apply stacks in order database → api (repos →
  push images → full apply) → jobs → sites.
- Terraform CLI may be absent locally — CI validates; state is local until
  the S3 backend blocks are filled in (required for infra.yaml).
