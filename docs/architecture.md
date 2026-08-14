# Architecture

The TCG catalog is a serverless system on AWS, organized as an NX monorepo.

```
  browser ─► CloudFront marketing site     browser ─► CloudFront app site     browser ─► CloudFront admin site
             (front door: pricing, CTAs)              (public catalog)                   (management, token login)
               └── default → S3 marketing               ├── default → S3 app               ├── default → S3 admin
                                                        └── /v1/* ───────┐                 └── /v1/* ──────┐
                                                                         ▼                                 │
                                                    API Gateway (HTTP API) ◄────────────────────────────────┘
                                                       │  route key → function (container-image Lambdas)
                                                       ├── …/games, /v1/auth/check ─► λ api-game-routes ┐
                                                       ├── …/sets… ─────────────────► λ api-set-routes  ├─► DynamoDB
                                                       └── …/cards… ────────────────► λ api-card-routes ┘  (single table)
                                                                                                             ▲
  EventBridge (schedule) ──► λ pokemon-price-updater ── price feed ──────────────────────────────────────────┘
```

All three sites are **static SvelteKit builds** (adapter-static) — no
server-side rendering at runtime; the only backend is API Gateway + Lambda.
App and admin call the API **same-origin** through their CloudFront
distribution — no CORS in the browser path; the direct API Gateway URL stays
available for services and local tooling. Every Lambda ships as a **Docker
image** (ECR); the same images run locally under docker-compose via the
bundled Runtime Interface Emulator and `tools/local-gateway`.

## Components

Two languages only: **TypeScript/Node for the web, Go for the Lambdas and
tooling.**

| Piece | Path (NX project) | Language | Why |
| --- | --- | --- | --- |
| HTTP Lambdas | `apps/api/game-routes` (`api-game-routes`), `apps/api/set-routes` (`api-set-routes`), `apps/api/card-routes` (`api-card-routes`) — one project + Go module + Dockerfile each | Go | Fast cold starts, tiny static binaries |
| Scheduled jobs | `apps/jobs/pokemon-price-updater` (`pokemon-price-updater`) | Go | Same runtime/store as the API |
| Marketing site | `apps/web/www` (`www`) | SvelteKit (static) | The product's front door: features, pricing, screenshots, CTA into the app |
| Public site | `apps/web/app` (`app`) | SvelteKit (static) | Catalog browser |
| Admin site | `apps/web/admin` (`admin`) | SvelteKit (static) | Catalog management, token login, **separate from the app** |
| Card-catalog store | `libs/card-catalog-store` (`card-catalog-store`) | Go | Domain + DynamoDB persistence for the card catalog; owns the single-table layout |
| Lambda HTTP plumbing | `libs/httpapi` (`httpapi`) | Go | Routing, bearer auth, error→status mapping |
| API client + types | `libs/api-client` (`api-client`, npm `@tcg/api-client`) | TypeScript | Shared by the SvelteKit sites |
| Design system | `libs/design-system` (`design-system`, npm `@tcg/design-system`) | TypeScript/Svelte | Tokens, base styles, shared components — one look & feel for all sites |
| Local API gateway | `tools/local-gateway` (`local-gateway`) | Go | docker-compose stand-in for API Gateway |
| Infrastructure | `infra/terraform` (`infra`) | HCL | Four independently-appliable stacks: `database`, `api`, `jobs`, `sites` |

**Sharing rules:** apps never import each other's code — shared code lives in
a lib directly under `libs/`, named specifically for its function (never
grouped by language, never generically named). ADR 0004–0006.

## Request flow

1. API Gateway matches a route key (`GET /v1/games/{game}/sets`) and proxies
   the request (payload v2) to the owning Lambda.
2. Each Lambda has an internal route table keyed by the same route key
   (`libs/httpapi.Route`). Non-GET routes require
   `Authorization: Bearer <API_TOKEN>`; reads are public. The admin site
   validates its login token via `POST /v1/auth/check`.
3. Handlers call `libs/card-catalog-store`, which speaks DynamoDB and returns
   domain errors (`ErrNotFound`, `ErrInvalid`, `ErrConflict`) that map to
   404/400/409. Unexpected errors log and return an opaque 500.

## Function split

One Lambda per aggregate (games / sets / cards) plus one per job, each with a
specific function name (`api-card-routes`, `pokemon-price-updater`):

- an aggregate shares a store surface and test fixtures — one binary per
  aggregate keeps cohesion without a 14-way build matrix;
- blast radius and metrics separate along the same lines operations think in;
- adding a route to an aggregate is a Terraform map entry, not a new function.

## Infrastructure stacks

`infra/terraform/stacks/{database,api,jobs,sites}` apply independently, wired
by `terraform_remote_state` (database → api/jobs → sites). See
[../infra/terraform/README.md](../infra/terraform/README.md).

## CI/CD (.github/workflows)

| Workflow | Status check | What |
| --- | --- | --- |
| `test.yaml` | `test` | unit tests only (`nx run-many -t test`) |
| `checks-nodejs.yaml` | `checks-nodejs` | `npm audit` (prod deps fail at high+) |
| `checks-golang.yaml` | `checks-golang` | `govulncheck` across all Go modules |
| `build.yaml` | `build` | Docker builds every Lambda image; pushes to ECR on main |
| `infra.yaml` | — | `terraform plan/apply` per stack, workflow_dispatch only |

Mark `test`, `checks-nodejs`, `checks-golang`, and `build` as **required
status checks** in branch protection so PRs can't merge without them.
`dependabot.yaml` keeps Actions, npm, and Go modules updated weekly.

## Local development

`docker compose up --build` runs the whole product (marketing :8082,
app :8080, admin :8081, API :3000, DynamoDB Local :8000) — see
[development.md](development.md). Unit tests never need Docker: the store is
tested against an in-memory fake (`libs/card-catalog-store/fake_dynamo_test.go`).

See also: [data-model.md](data-model.md), [api.md](api.md), and the decision
records under [decisions/](decisions/).
