# TCG Catalog

A serverless trading-card catalog for multiple TCGs (Pokémon, Magic, Lorcana,
sports cards): games → sets → cards → variants with current prices.

Everything runs on AWS — container-image Go Lambdas behind an API Gateway
HTTP API, a DynamoDB single table, and three static SvelteKit sites
(marketing, app, admin) on S3 + CloudFront — provisioned by Terraform in four
independent stacks and organized as an NX monorepo. Two languages:
**TypeScript for the web, Go for the Lambdas.**

```
apps/
  api/game-routes/       λ api-game-routes  ┐ one NX project, Go module,
  api/set-routes/        λ api-set-routes   ├ and Dockerfile per function
  api/card-routes/       λ api-card-routes  ┘
  jobs/pokemon-price-updater/  λ scheduled price refresh (EventBridge)
  marketing/             product front door: features, pricing, CTA → app (SvelteKit)
  app/                   public catalog browser (SvelteKit)
  admin/                 catalog management — token login, separate site (SvelteKit)
libs/                    shared modules, flat, named for their function
  card-catalog-store/    card-catalog domain + DynamoDB persistence (Go)
  httpapi/               Lambda routing/auth/error plumbing (Go)
  api-client/            @tcg/api-client — API types + client (TS)
tools/
  local-gateway/         docker-compose stand-in for API Gateway (Go)
  scripts/               DynamoDB Local + ECR push helpers
infra/
  terraform/stacks/      database · api · jobs · sites — applied independently
.github/                 test / checks-nodejs / checks-golang / build / infra + dependabot
docs/                    Architecture, data model, API reference, ADRs
MEMORY.md                Log of Claude requests and outcomes
```

## Quick start

```sh
npm install
npx nx run-many -t test        # all unit tests (Go + Vitest)
docker compose up --build      # the ENTIRE product locally:
                               #   marketing :8082 · app :8080 · admin :8081
                               #   API :3000 · DynamoDB :8000
npx nx dev app                 # frontend dev server against the compose API
```

## API in one breath

Public reads, bearer-token writes (admin logs in with the token via
`POST /v1/auth/check`):

```sh
curl -s "$API/v1/games"
curl -s "$API/v1/games/pokemon/sets?query=151"
curl -s "$API/v1/games/pokemon/cards?query=zapdos"
curl -s "$API/v1/games/pokemon/cards/501773"       # TCGplayer id or GUID
```

Full reference: [docs/api.md](docs/api.md).

## CI / merge gates

PRs must pass `test`, `checks-nodejs`, `checks-golang`, and `build` (mark
them required in branch protection). `build.yaml` publishes Lambda images to
ECR on main; `infra.yaml` runs Terraform per stack on demand; Dependabot
updates Actions/npm/Go weekly.

## More

- [docs/development.md](docs/development.md) — day-to-day workflows
- [docs/architecture.md](docs/architecture.md) / [docs/data-model.md](docs/data-model.md)
- [infra/terraform/README.md](infra/terraform/README.md) — deploying stack by stack
- [docs/decisions/](docs/decisions/) — why things are the way they are

## History

This began as a single Go service (PostgreSQL + GraphQL + gRPC), was
converted to a serverless monorepo, and has been reshaped stepwise since —
the reasoning lives in [docs/decisions/](docs/decisions/) and the request log
in [MEMORY.md](MEMORY.md).
