# 0005 — App/admin split, container-image Lambdas, flat libs

Date: 2026-08-13 · Status: accepted

## Context

Owner directives: the website splits into a public site and a separate
management site with limited logins; Lambdas get specific function names
(`api-card-routes`, `pokemon-price-updater`) and deploy as Docker images;
libs must not be grouped by language; docker-compose must run the entire
site locally.

## Decisions

**Two SPAs.** `apps/web/app` (public, read-only) and `apps/web/admin` (catalog
management) — separate NX projects, separate S3 buckets and CloudFront
distributions. "Limited logins" is implemented with the existing trust
model: the admin signs in with the API bearer token, validated by the new
`POST /v1/auth/check` (204/401) and held in sessionStorage. If real user
accounts are ever needed, Cognito can replace the token check behind the
same login screen.

**Container-image Lambdas.** Every function has a Dockerfile (repo-root
build context, `golang:1.26` build stage → `public.ecr.aws/lambda/provided:al2023`,
arm64). Terraform provisions one ECR repo per function
(`modules/lambda-function`), `tools/scripts/push-images.sh` builds/pushes,
`image_tag` selects the deployed tag. Why: identical artifact locally and in
AWS (the base image bundles the Runtime Interface Emulator, which is what
makes the compose stack possible), no zip/runtime coupling, image lifecycle
policies. Cost: first deploy needs repos → push → apply ordering.

**Function naming.** `api-<aggregate>-routes` for the HTTP functions,
`<game>-<job>` for jobs. New `pokemon-price-updater`: EventBridge-scheduled,
pulls quotes from a pluggable JSON feed (`PRICE_API_URL`), writes variant
prices through the new `Catalog.SetCardPrices` (which finally gives embedded
variants a write path). Disabled by default until a feed is configured.

**Flat libs, named by module**: `libs/catalog` (Go), `libs/httpapi` (Go),
`libs/api-client` (TS, npm `@tcg/api-client`). The language is an
implementation detail of the module, not a taxonomy level.

**Local stack.** `docker compose up --build` runs DynamoDB Local (+ table
init), the three route Lambdas as real Lambda containers,
`tools/local-gateway` (a ~200-line stand-in for API Gateway that wraps HTTP
into invoke events; its route table mirrors `apigateway.tf`), and nginx
containers serving the built app/admin with `/v1` proxied to the gateway.
The price updater sits behind a `jobs` profile and is invoked manually.

## Consequences

- Route changes now touch three places: the Lambda's handler map,
  `apigateway.tf`, and `tools/local-gateway/main.go` (compose).
- Deploys require Docker; plain `go build` remains the CI compile/test path.
- Admin and app can diverge freely (different design, dependencies, cadence).
