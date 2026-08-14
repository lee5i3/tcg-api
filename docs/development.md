# Development guide

## Prerequisites

Two languages only — Node for the web, Go for the Lambdas:

- Node 20+ / npm (NX, the three SvelteKit sites, `@tcg/api-client`)
- Go 1.26+ (Lambdas, card-catalog-store, local-gateway)
- Docker (compose stack, Lambda images)
- Terraform 1.6+ and the AWS CLI (deploys only)

## First-time setup

```sh
npm install          # NX + site deps (npm workspaces)
```

## Everyday commands

```sh
npx nx test card-catalog-store   # Go store tests (in-memory DynamoDB fake, no Docker)
npx nx test httpapi              # shared Lambda plumbing tests
npx nx test api-game-routes      # one Lambda's handler tests (api-set-routes, api-card-routes …)
npx nx test pokemon-price-updater
npx nx test local-gateway        # the compose API-gateway stand-in
npx nx test api-client           # shared TS API client tests
npx nx test app                  # public site (admin, www likewise)
npx nx run-many -t test          # everything
npx nx run-many -t lint          # go vet + gofmt, svelte-check/tsc, terraform fmt
npx nx build app                 # static SvelteKit bundle → apps/web/app/dist (admin, www likewise)
npx nx dev app                   # dev server (admin :5174)
npx nx graph                     # dependency graph
```

Go also works directly from the repo root (`go.work` wires the seven
modules). Patterns must be rooted in a module (`./...` across modules is
rejected by this Go toolchain):

```sh
go test ./libs/card-catalog-store/... ./libs/httpapi/... ./tools/local-gateway/... \
        ./apps/api/game-routes/... ./apps/api/set-routes/... ./apps/api/card-routes/... \
        ./apps/jobs/pokemon-price-updater/...
```

## Running the entire site locally

```sh
docker compose up --build
```

| URL | What |
| --- | --- |
| http://localhost:8082 | marketing site |
| http://localhost:8080 | public app |
| http://localhost:8081 | admin (login token: `local-dev-token`) |
| http://localhost:3000 | REST API (`tools/local-gateway` → the real Lambda images via their Runtime Interface Emulator) |
| http://localhost:8000 | DynamoDB Local (table auto-created) |

Seed data through the admin UI, or curl:

```sh
curl -X POST http://localhost:3000/v1/games/pokemon/sets \
  -H 'Authorization: Bearer local-dev-token' -H 'Content-Type: application/json' \
  -d '{"key":"sv3pt5","name":"151","releaseDate":"2023-09-22"}'
```

The price updater is behind a compose profile (`--profile jobs`); invoke it
manually with a POST to its RIE port (see docker-compose.yaml comments).

**Frontend inner loop:** skip the image rebuilds — keep the compose stack up
and run `npx nx dev app` (or `admin`) with `VITE_API_URL=http://localhost:3000`;
the gateway sends permissive CORS headers for exactly this case. Unit tests
never need Docker — the Go store tests run against an in-memory fake.

## CI and merge gates

Every PR must pass the required status checks: `test` (unit tests),
`checks-nodejs` (npm audit), `checks-golang` (govulncheck), `build` (Docker
images build). `build.yaml` pushes images to ECR on main; `infra.yaml`
(workflow_dispatch) plans/applies one Terraform stack at a time. Dependabot
(`.github/dependabot.yaml`) opens weekly update PRs for Actions, npm, and Go.

## Conventions

- **Go**: standard library first; table-driven tests; wrap domain errors with
  `%w` around `catalog.ErrNotFound/ErrInvalid/ErrConflict`; `gofmt` is CI-enforced.
- **Every Lambda declares a narrow `Store` interface** of just the catalog
  methods it uses — handler tests fake that interface, store tests fake DynamoDB.
- **Web**: SvelteKit + TypeScript strict, static builds only (`dist/`);
  API access exclusively through `@tcg/api-client`.
- **Libs are named for their function** (`card-catalog-store`, not `catalog`)
  and live flat under `libs/` — never grouped by language.
- **Terraform**: four stacks under `infra/terraform/stacks/`, applied
  independently; `terraform fmt`; no hardcoded secrets.
- **Docs**: decisions get a record in [docs/decisions/](decisions/); update
  [CLAUDE.md](../CLAUDE.md) and [MEMORY.md](../MEMORY.md) when requirements
  change; keep [data-model.md](data-model.md) in sync with
  `libs/card-catalog-store/dynamo.go`.

## Deploying

See [infra/terraform/README.md](../infra/terraform/README.md).
