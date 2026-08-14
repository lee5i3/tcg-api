# @tcg/app — catalog browser

SvelteKit (Svelte 5) single-page app for browsing the TCG catalog: games,
sets, card grids with lowest variant prices, name search, and card detail
pages with a variant price table.

**Sign-in required**: the catalog is gated behind user accounts. Opening the
app lands on `/login` (with a link to `/register` for creating an account);
`/login` and `/register` are the only public routes. On success the user JWT
from `POST /v1/auth/login` / `POST /v1/auth/register` is kept in
localStorage (`tcg-app-token`), revalidated in the background via
`GET /v1/auth/me` on app start, and sent as an `Authorization: Bearer`
header on API requests. Signing out (header button) clears the token.

It is a **fully static build** (`@sveltejs/adapter-static`, `ssr = false`,
SPA fallback `index.html`) — all data is fetched client-side from the REST
API (`/v1/...`); the auth guard runs client-side in the root layout.

## Commands (via NX or npm workspaces)

- `nx dev app` — Vite dev server
- `nx build app` — static build to `apps/app/dist/`
- `nx test app` — Vitest (jsdom + @testing-library/svelte)
- `nx lint app` — svelte-check

## Configuration

- `VITE_API_URL` — base URL of the catalog API. Leave empty for same-origin
  `/v1/...` requests (production: nginx/CloudFront proxies to the gateway).
  See `.env.example`.

## Docker

`docker build -f apps/app/Dockerfile .` from the repo root: builds the
static site, then serves it with nginx (SPA fallback; `/v1/` and `/healthz`
proxied to `${GATEWAY_URL}`).
