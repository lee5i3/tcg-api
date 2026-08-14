<script lang="ts">
  import "@tcg/design-system/tokens.css";
  import "@tcg/design-system/base.css";
  import "../app.css";
  import { Button, SiteHeader } from "@tcg/design-system";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { auth } from "$lib/auth.svelte";
  import { guardTarget } from "$lib/guard";
  import type { LayoutProps } from "./$types";

  let { children }: LayoutProps = $props();

  // Client-side guard for every route: unauthenticated -> /login,
  // authenticated visitors of /login -> dashboard.
  const target = $derived(guardTarget(auth.token, page.url.pathname));

  // /login renders bare (full-viewport split-screen, brand lives in-page).
  const isAuthPage = $derived(page.url.pathname === "/login");

  $effect(() => {
    if (target !== null) {
      void goto(target, { replaceState: true });
    }
  });
</script>

{#if isAuthPage}
  {#if target === null}
    {@render children()}
  {/if}
{:else}
  <div class="admin">
    <SiteHeader badge="Admin">
      {#if auth.token}
        <Button variant="secondary" size="sm" type="button" onclick={() => auth.logout()}>
          Log out
        </Button>
      {/if}
    </SiteHeader>
    <main class="admin-main">
      {#if target === null}
        {@render children()}
      {:else}
        <p class="status">Redirecting…</p>
      {/if}
    </main>
  </div>
{/if}
