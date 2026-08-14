<script lang="ts">
  // Optional social sign-in block for /login and /register: an
  // "or continue with" divider followed by one full-width outlined link
  // per provider the API is configured for. When the providers endpoint
  // returns nothing (or fails), the whole section renders nothing —
  // social sign-in is strictly optional.
  import { onMount } from "svelte";
  import { Button } from "@tcg/design-system";
  import type { AuthProvider } from "@tcg/api-client";
  import { api } from "$lib/api";
  import { sanitizeRedirect } from "$lib/auth.svelte";

  let { redirect = null }: { redirect?: string | null } = $props();

  // Fixed display order for the well-known providers; anything else keeps
  // its server order after them (Array.prototype.sort is stable).
  const ORDER = ["google", "facebook", "apple"];
  const rank = (p: AuthProvider): number => {
    const i = ORDER.indexOf(p.id);
    return i === -1 ? ORDER.length : i;
  };

  let providers = $state<AuthProvider[]>([]);

  onMount(() => {
    api
      .providers()
      .then((list) => {
        providers = [...list].sort((a, b) => rank(a) - rank(b));
      })
      .catch(() => {
        // Leave the list empty: no section, no error — social is optional.
      });
  });

  // Browser-navigation target: the backend 302s on to the provider.
  const startUrl = (p: AuthProvider): string =>
    `/v1/auth/oauth/${encodeURIComponent(p.id)}/start?redirect=${encodeURIComponent(
      sanitizeRedirect(redirect),
    )}`;
</script>

{#if providers.length > 0}
  <div class="auth-divider" aria-hidden="true"><span>or continue with</span></div>
  <div class="auth-social">
    {#each providers as provider (provider.id)}
      <!-- data-sveltekit-reload: /v1/... is the API, not an app route, so the
           router must let the browser navigate for the OAuth round-trip. -->
      <Button
        variant="inverse"
        class="auth-btn"
        href={startUrl(provider)}
        data-sveltekit-reload
      >
        {#if provider.id === "google"}
          <svg class="auth-social-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="#4285F4"
              d="M23.52 12.27c0-.85-.08-1.66-.22-2.45H12v4.64h6.46a5.53 5.53 0 0 1-2.4 3.62v3.01h3.88c2.27-2.09 3.58-5.17 3.58-8.82z"
            />
            <path
              fill="#34A853"
              d="M12 24c3.24 0 5.96-1.08 7.94-2.91l-3.88-3.01c-1.07.72-2.45 1.15-4.06 1.15-3.12 0-5.77-2.11-6.71-4.95H1.28v3.11A12 12 0 0 0 12 24z"
            />
            <path
              fill="#FBBC05"
              d="M5.29 14.28A7.2 7.2 0 0 1 4.91 12c0-.79.14-1.56.38-2.28V6.61H1.28a12 12 0 0 0 0 10.78l4.01-3.11z"
            />
            <path
              fill="#EA4335"
              d="M12 4.77c1.76 0 3.34.61 4.58 1.79l3.44-3.43A12 12 0 0 0 1.28 6.61l4.01 3.11C6.23 6.88 8.88 4.77 12 4.77z"
            />
          </svg>
        {:else if provider.id === "facebook"}
          <svg class="auth-social-icon" viewBox="0 0 24 24" aria-hidden="true">
            <defs>
              <clipPath id="fb-circle"><circle cx="12" cy="12" r="12" /></clipPath>
            </defs>
            <circle cx="12" cy="12" r="12" fill="#1877F2" />
            <path
              clip-path="url(#fb-circle)"
              fill="#ffffff"
              d="M16.67 15.47 17.2 12h-3.33V9.75c0-.95.47-1.87 1.96-1.87h1.51V4.92s-1.37-.24-2.69-.24c-2.74 0-4.53 1.66-4.53 4.67V12H7.08v3.47h3.04V24a12.06 12.06 0 0 0 3.75 0v-8.53h2.8z"
            />
          </svg>
        {:else if provider.id === "apple"}
          <svg class="auth-social-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="#ffffff"
              d="M12.15 6.9c-.95 0-2.42-1.08-3.96-1.04-2.04.02-3.91 1.18-4.96 3.01-2.12 3.67-.55 9.1 1.52 12.09 1.01 1.45 2.21 3.09 3.79 3.03 1.52-.06 2.09-.99 3.94-.99 1.83 0 2.35.99 3.96.95 1.64-.03 2.68-1.48 3.68-2.95 1.15-1.69 1.63-3.32 1.66-3.41-.04-.02-3.18-1.22-3.22-4.86-.03-3.04 2.48-4.49 2.6-4.56-1.43-2.09-3.62-2.32-4.39-2.38-2-.16-3.68 1.09-4.62 1.09zm3.38-3.07c.84-1.01 1.4-2.42 1.25-3.83-1.21.05-2.66.81-3.53 1.82-.78.9-1.46 2.34-1.28 3.71 1.34.11 2.72-.68 3.56-1.7z"
            />
          </svg>
        {/if}
        Continue with {provider.label}
      </Button>
    {/each}
  </div>
{/if}

<style>
  .auth-social {
    display: flex;
    flex-direction: column;
    gap: var(--ds-space-3);
  }

  .auth-social-icon {
    width: 1.125rem;
    height: 1.125rem;
    flex-shrink: 0;
  }
</style>
