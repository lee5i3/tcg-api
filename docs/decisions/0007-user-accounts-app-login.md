# 0007 — User accounts: the app starts at a login page

Date: 2026-08-14 · Status: accepted

## Context

Owner directive: opening the app must start with a login page that also
offers registration. Until now the app was an open catalog browser and the
only credential in the system was the admin `API_TOKEN`.

## Decision

**Real end-user accounts, minimal shape:**

- `libs/user-accounts` (named for its function): register / authenticate /
  fetch, in the same DynamoDB single table under `USER#{id}` +
  `USEREMAIL#{email}` guard items (the established uniqueness pattern).
  Passwords are bcrypt hashes; wrong email and wrong password are
  indistinguishable in both response and (roughly) timing.
- `apps/api/auth-routes` (`api-auth-routes`), the fourth HTTP Lambda:
  public `POST /v1/auth/register` + `/login` (which required teaching
  `libs/httpapi` a public-routes dispatch — its old rule was "all non-GET
  needs the admin token"), `GET /v1/auth/me`, and `POST /v1/auth/check`
  (moved here from game-routes so all of `/v1/auth/*` has one owner).
- **Sessions are stateless JWTs** (HS256, `USER_JWT_SECRET`, 7-day expiry) —
  no session table, no logout invalidation server-side; sign-out is the app
  discarding the token. Good enough until there's a revocation requirement.
- The app gates all routes client-side: token in localStorage, validated via
  `/v1/auth/me` on load, `/login` and `/register` the only public pages.

**What this deliberately does not change:** the catalog READ endpoints stay
public API surface (services and the marketing pipeline depend on them) —
the login wall is the app's UX, not an API-wide lockdown. Catalog writes
still require the admin token; user JWTs grant no write access. If reads
must later be locked down or users get per-account data (collections,
watchlists), extend `httpapi` to verify user JWTs on those routes.

## Consequences

- Adding an auth route = the same three places as any route (handler map,
  apigateway.tf, local-gateway) — unchanged rule.
- `USER_JWT_SECRET` joins `api_token` as a required production secret;
  rotating it signs everyone out.
- Registration still has no email verification or payment hook — the
  marketing site's signup TODO can now point at `/register`.
