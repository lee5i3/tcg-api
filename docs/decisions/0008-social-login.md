# 0008 — Social login (Google, Facebook, Apple)

Date: 2026-08-14 · Status: accepted · Extends 0007

## Context

Owner directive: the app's login page offers social sign-in with Google,
Facebook, and Apple, alongside the email/password accounts from ADR 0007.

## Decision

**Self-hosted OAuth 2.0 / OIDC authorization-code flow in `api-auth-routes`**
(no Cognito — it would add a second user store next to ours for three
standard integrations):

- `GET /v1/auth/providers` returns only providers whose credentials are
  configured, and the sites render buttons from it — **no dead buttons**.
- `start` 302s to the provider with a signed state JWT (HS256, 10-min TTL,
  distinct issuer — state and session tokens verify in one direction only).
- `callback` exchanges the code server-side, reads the identity from the
  id_token (Google/Apple) or the Graph API (Facebook), and 302s back to the
  app with our session JWT **in the URL fragment** (kept out of access logs).
  Apple's `form_post` response mode makes the callback a POST — both methods
  are routed. id_tokens are decoded without JWKS verification: they arrive
  directly from the provider's token endpoint over TLS, never from a client.
- **Account linking by verified email** (`user-accounts.FindOrCreateOAuthUser`):
  known identity → its account; unknown identity with a known email → linked
  to that account (all three providers verify emails); otherwise a new
  social-only account with no password. Identity uniqueness uses the same
  transactional guard-item pattern as emails (`USERIDENT#{provider}#{sub}`).
- redirect_uri is `{APP_URL}/v1/auth/oauth/{provider}/callback` — through the
  app's CloudFront distribution, so the whole round trip is same-origin.
  `app_url` is a Terraform variable on the api stack (set after the first
  sites apply; a variable rather than remote state to avoid an api↔sites
  dependency cycle).

## Consequences

- Six new Terraform variables (3× id+secret) + `app_url`; Apple's "secret"
  is an expiring ES256 JWT the operator must rotate (≤6 months).
- Locally, buttons appear only if provider credentials are exported before
  `docker compose up` — with real (test-app) credentials the full flow works
  against `http://localhost:8080`.
- No session revocation for social logins either — same stateless-JWT
  trade-off as ADR 0007.
