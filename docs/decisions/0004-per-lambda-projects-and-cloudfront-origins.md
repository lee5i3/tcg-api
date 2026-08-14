# 0004 — One project per Lambda; CloudFront fronts web + API

Date: 2026-08-13 · Status: accepted

## Context

After the initial conversion (ADR 0001–0003) the three Lambdas lived in one
Go module (`apps/api`) sharing an `internal/httpapi` package, and the website
called the API Gateway URL cross-origin via CORS.

## Decision

**Each Lambda is its own NX project and Go module** — `apps/api/games`,
`apps/api/sets`, `apps/api/cards` (projects `api-games`, `api-sets`,
`api-cards`) — and shared code lives in named libraries only:

- `libs/go/catalog` (`catalog-go`) — domain + DynamoDB store
- `libs/go/httpapi` (`httpapi-go`) — Lambda HTTP plumbing (routing, auth, error mapping)
- `libs/ts/api-client` (`api-client`, npm `@tcg/api-client`) — API types + client shared by web apps

Apps never import each other's code; anything shared must first be promoted
to a lib. This gives per-function builds/caching (`nx build api-cards`
rebuilds one binary), independent dependency surfaces, and NX affected-graph
precision.

**The CloudFront distribution has two origins**: S3 (SPA) as default, and
the API Gateway HTTP API under `/v1/*` + `/healthz` (CachingDisabled +
AllViewerExceptHostHeader policies). One URL serves the product; the browser
calls the API same-origin, so CORS drops out of the browser path. SPA
routing moved from distribution-wide 403/404→index.html error responses
(which would have swallowed API errors) to a CloudFront Function on the web
behavior that rewrites extensionless paths to `/index.html`.

## Consequences

- The web client defaults its base URL to `""` (same-origin); `VITE_API_URL`
  remains an override for local dev against a remote API.
- The direct `api_endpoint` output still exists for services (importer),
  where CORS is irrelevant.
- Five Go modules in `go.work`; each Lambda's `go.mod` carries `replace`
  directives to the libs so it also builds standalone.
- New shared code has an explicit home decision: which lib does it belong to?
