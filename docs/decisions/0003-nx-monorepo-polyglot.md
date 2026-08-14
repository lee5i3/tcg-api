# 0003 — NX monorepo, language per job

Date: 2026-08-13 · Status: accepted

## Context

The repo needed to grow from one Go service to lambdas + website + importer +
infrastructure, with room for more apps.

## Decision

NX drives every project through a uniform target vocabulary
(`build`/`test`/`lint`) with caching; each project uses the language that
fits, wired in the lightest way that works:

- **Go** (`apps/api`, `libs/go/catalog`) — Lambda cold starts and static
  binaries; projects are `project.json` + `nx:run-commands` over the plain Go
  toolchain, joined by `go.work` (no community Go plugin to version-chase;
  `apps/api/go.mod` carries a `replace` so it also builds without workspace mode).
- **TypeScript/React** (`apps/web`) — npm workspace member; NX infers targets
  from its package.json scripts.
- **Python** (`tools/importer`) — data wrangling; self-contained venv behind
  `run-tests.sh`, driven by `project.json`.
- **Terraform** (`infra/terraform`) — `lint`/`plan`/`apply` targets.

## Consequences

- One entry point for everything: `npx nx run-many -t test` / `-t lint`.
- `npx nx graph` shows cross-language dependencies (api ← catalog-go,
  infra ← api) via `implicitDependencies`.
- No NX plugin lock-in for Go/Python — the executors are plain shell
  commands, so the toolchains stay independently upgradable.
- Contributors need three toolchains installed, but only for the projects
  they touch (NX skips targets whose inputs didn't change).
