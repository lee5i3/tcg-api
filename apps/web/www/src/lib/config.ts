/**
 * Absolute URL of the public catalog app. Every CTA on the marketing site
 * points here for now.
 *
 * TODO(signup): registration/payment is not built yet. When the signup /
 * checkout flow exists, route the "Get started" and per-tier CTAs to it
 * instead of the app root (e.g. `${appUrl()}/signup?plan=<tier id>`).
 */
export function appUrl(): string {
  return import.meta.env.PUBLIC_APP_URL ?? "https://app.example.com";
}
