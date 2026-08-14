# @tcg/admin — catalog admin

SvelteKit (Svelte 5) single-page app for managing the TCG catalog:

- Token login via `POST /v1/auth/check` (kept in `sessionStorage` for the
  browser session, all routes guarded client-side, logout in the header).
- Games: list + create.
- Sets per game: search, create, edit, delete (with confirm; shows how many
  cards were deleted).
- Cards per set: create, edit, delete.
- API errors (e.g. 409 conflicts) are shown inline next to the forms.

It is a **fully static build** (`@sveltejs/adapter-static`, `ssr = false`,
SPA fallback `index.html`) with a deliberately distinct dark/amber identity.

## Commands (via NX or npm workspaces)

- `nx dev admin` — Vite dev server
- `nx build admin` — static build to `apps/admin/dist/`
- `nx test admin` — Vitest (jsdom + @testing-library/svelte)
- `nx lint admin` — svelte-check

## Configuration

- `VITE_API_URL` — base URL of the catalog API. Leave empty for same-origin
  `/v1/...` requests. See `.env.example`.

## Docker

`docker build -f apps/admin/Dockerfile .` from the repo root: builds the
static site, then serves it with nginx (SPA fallback; `/v1/` and `/healthz`
proxied to `${GATEWAY_URL}`).
