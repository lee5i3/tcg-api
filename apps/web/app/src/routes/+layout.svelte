<script lang="ts">
  import "@tcg/design-system/tokens.css";
  import "@tcg/design-system/base.css";
  import "../app.css";
  import { Button, SiteHeader } from "@tcg/design-system";
  import type { Snippet } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { auth, clearSession, initAuth, validateSession } from "$lib/auth.svelte";

  let { children }: { children: Snippet } = $props();

  // The catalog requires a signed-in user; only the auth pages (and the
  // OAuth landing pad, which is busy storing the session) are public.
  const publicRoutes = ["/login", "/register", "/auth/callback"];
  const isPublic = $derived(publicRoutes.includes(page.url.pathname));

  // Restore a persisted token before anything renders.
  initAuth();

  // Validate the restored token in the background on app start; a 401
  // clears the session and the guard below redirects to /login.
  $effect(() => {
    if (auth.token && !auth.user) {
      void validateSession();
    }
  });

  // Client-side guard: unauthenticated users are sent to /login from any
  // non-public route, remembering where they wanted to go.
  $effect(() => {
    if (!auth.isAuthenticated && !isPublic) {
      const returnTo = page.url.pathname + page.url.search;
      const target =
        returnTo === "/" ? "/login" : `/login?redirect=${encodeURIComponent(returnTo)}`;
      void goto(target, { replaceState: true });
    }
  });

  function signOut(): void {
    clearSession();
    void goto("/login");
  }
</script>

{#if isPublic}
  <!-- Auth pages are a full-viewport split-screen: no SiteHeader — the
       brand lockup lives inside the page's left column. -->
  {@render children()}
{:else}
  <div class="app">
    <SiteHeader>
      {#if auth.isAuthenticated}
        {#if auth.user}
          <span class="header-user">{auth.user.name || auth.user.email}</span>
        {/if}
        <Button variant="secondary" size="sm" type="button" onclick={signOut}>Sign out</Button>
      {/if}
    </SiteHeader>
    <main class="app-main">
      {#if auth.isAuthenticated}
        {@render children()}
      {/if}
    </main>
  </div>
{/if}
