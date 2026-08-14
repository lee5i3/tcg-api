# 0006 — SvelteKit static sites, two languages, CI gates, stacked Terraform

Date: 2026-08-13 · Status: accepted · Supersedes parts of 0003 (Python) and 0005 (React sites, monolithic Terraform)

## Context

Owner directives: all websites in SvelteKit hosted as static builds (API
stays API Gateway + Lambda); only Node.js (web) and Go (Lambdas) — Python
dropped; a marketing site as the product's front door; PR-gating CI for
tests, security checks, and image builds; Terraform split into independently
appliable components; shared libraries named specifically for their function.

## Decisions

**SvelteKit, adapter-static, `dist/` output** for all three sites —
`marketing` (new: hero/features/pricing/screenshots/FAQ, CTAs to the app;
signup/checkout intentionally not built — a marked TODO hooks a future
Stripe/Cognito flow), `app` (public catalog), `admin` (token login,
management). Client-side fetching only; the API backend is unchanged.
`@tcg/api-client` stays the single API surface for all sites.

**Two languages.** Python is gone; the `tools/importer` CLI was deleted with
it (bulk loading = admin UI or scripted curl against the write API; a Go
importer can be added if real need returns).

**Specific lib names.** `libs/catalog` → `libs/card-catalog-store` (domain +
DynamoDB persistence for the card catalog). Go package name inside remains
`catalog` — import path carries the specificity; renaming the package would
churn every file for no behavioral gain.

**CI as merge gates** (all named jobs = required status checks: `test`,
`checks-nodejs`, `checks-golang`, `build`):
- `test.yaml` — unit tests only.
- `checks-nodejs.yaml` — `npm audit` (prod deps fail at high+; dev-dep audit
  informational), plus a weekly schedule.
- `checks-golang.yaml` — `govulncheck` across all seven Go modules, weekly too.
- `build.yaml` — builds all four Lambda images on every PR (arm64 via
  buildx/QEMU, GHA cache); pushes SHA+latest tags to ECR on main/dispatch
  via GitHub OIDC (`AWS_ROLE_ARN`).
- `infra.yaml` — workflow_dispatch `terraform plan/apply` per stack.
- `dependabot.yaml` — weekly Actions/npm/gomod updates, grouped.
Branch protection itself is repo settings, not code — the required-check
names are documented in architecture.md.

**Stacked Terraform.** `infra/terraform/stacks/{database,api,jobs,sites}`,
each with its own state, wired by `terraform_remote_state`
(database → api/jobs → sites). Shared `modules/` unchanged. Local-backend
paths work out of the box; CI applies require the S3 backend blocks to be
filled in.

## Consequences

- A schema change applies in minutes (database stack) without touching
  CloudFront; site changes can't break the API stack.
- Bulk data loading has no dedicated CLI right now.
- SvelteKit static means no SSR/edge rendering; if SEO on the catalog pages
  ever matters, revisit with adapter + a rendering host (marketing pages are
  prerendered and fine).
- Four more moving parts in .github — but every gate is scoped and fast.
