# @tcg/marketing — marketing site

SvelteKit (Svelte 5) marketing/landing site for TCG Catalog — the future
root domain. **Fully prerendered** (`@sveltejs/adapter-static`,
`prerender = true`): the build emits plain static HTML, no client data
fetching and no API dependency.

Sections: hero (collection-value angle), features, screenshots (abstract
inline-SVG mockups — deliberately not fabricated screenshots), pricing
(3 placeholder tiers: Free / Collector $9/mo / Pro $29/mo), FAQ, footer.

All CTAs ("Open the app", "Get started") link to the public app.
Registration/payment is not built yet — see `TODO(signup)` in
`src/lib/config.ts` for where a signup/checkout flow will hook in.

## Commands (via NX or npm workspaces)

- `nx dev marketing` — Vite dev server
- `nx build marketing` — prerendered static build to `apps/marketing/dist/`
- `nx test marketing` — Vitest (jsdom + @testing-library/svelte)
- `nx lint marketing` — svelte-check

## Configuration

- `PUBLIC_APP_URL` — absolute URL of the public catalog app, baked into the
  prerendered pages at build time. Defaults to the placeholder
  `https://app.example.com`. See `.env.example`.

## Docker

`docker build -f apps/marketing/Dockerfile .` from the repo root: builds the
static site, then serves it with nginx (plain static serving with a
prerendered `404.html`; no API proxy).
