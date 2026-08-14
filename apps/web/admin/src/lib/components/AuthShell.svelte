<script lang="ts">
  // Admin login chrome: the same dark split-screen the app's auth pages use
  // (design-system AuthSplit), with an admin-flavored right panel.
  import { AuthSplit, Badge, BrandMark } from "@tcg/design-system";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  // Static, tasteful showcase numbers — mirrors the app login's stat card.
  const rows = [
    { name: "Games", value: "4" },
    { name: "Sets", value: "128" },
    { name: "Cards", value: "24,310" },
    { name: "Variant prices", value: "58,904" },
  ];
</script>

<AuthSplit>
  {#snippet brand()}
    <div class="auth-brand">
      <span class="auth-brand-row">
        <BrandMark />
        <Badge variant="neutral">Admin</Badge>
      </span>
      <p class="auth-tagline">Catalog administration</p>
    </div>
  {/snippet}

  {@render children()}

  {#snippet aside()}
    <div class="auth-pitch">
      <h2 class="auth-pitch-title">Run the catalog.</h2>
      <p class="auth-pitch-copy">
        Games, sets, cards, and variant prices — managed in one place, live on
        every site the moment you save.
      </p>
      <div class="auth-ticker">
        <div class="auth-ticker-head">
          <span class="auth-ticker-dot" aria-hidden="true"></span>
          <span class="auth-ticker-label">Catalog at a glance</span>
        </div>
        <ul class="auth-ticker-rows">
          {#each rows as row (row.name)}
            <li class="auth-ticker-row">
              <span class="auth-ticker-name">{row.name}</span>
              <span class="auth-ticker-value ds-tabular">{row.value}</span>
            </li>
          {/each}
        </ul>
      </div>
    </div>
  {/snippet}
</AuthSplit>

<style>
  .auth-brand {
    display: flex;
    flex-direction: column;
    gap: var(--ds-space-1);
  }

  .auth-brand-row {
    display: flex;
    align-items: center;
    gap: var(--ds-space-3);
    color: var(--ds-color-on-dark);
  }

  .auth-tagline {
    margin: 0;
    padding-left: 2.35em; /* clear the brand tile, align with the wordmark */
    font-size: var(--ds-text-xs);
    color: var(--ds-color-on-dark-muted);
  }

  .auth-pitch {
    max-width: 30rem;
    margin: 0 auto;
  }

  .auth-pitch-title {
    margin: 0 0 var(--ds-space-3);
    font-size: var(--ds-text-2xl);
    letter-spacing: var(--ds-tracking-tighter);
    color: #ffffff;
  }

  .auth-pitch-copy {
    margin: 0 0 var(--ds-space-6);
    color: var(--ds-color-on-dark-muted);
    font-size: var(--ds-text-md);
  }

  .auth-ticker {
    background: var(--ds-color-dark-raised);
    border: 1px solid var(--ds-color-dark-line);
    border-radius: var(--ds-radius-lg);
    box-shadow: var(--ds-shadow-lg);
    padding: var(--ds-space-4) var(--ds-space-5);
  }

  .auth-ticker-head {
    display: flex;
    align-items: center;
    gap: var(--ds-space-2);
    padding-bottom: var(--ds-space-3);
    border-bottom: 1px solid var(--ds-color-dark-line);
  }

  .auth-ticker-dot {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: var(--ds-radius-full);
    background: #4ade80;
    box-shadow: 0 0 0 3px rgba(74, 222, 128, 0.2);
  }

  .auth-ticker-label {
    font-size: var(--ds-text-sm);
    font-weight: var(--ds-weight-medium);
    color: var(--ds-color-on-dark-muted);
  }

  .auth-ticker-rows {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .auth-ticker-row {
    display: flex;
    align-items: baseline;
    gap: var(--ds-space-3);
    padding: var(--ds-space-3) 0;
    border-bottom: 1px solid rgba(148, 163, 184, 0.12);
    font-family: var(--ds-font-mono);
    font-size: var(--ds-text-sm);
  }

  .auth-ticker-row:last-child {
    border-bottom: none;
    padding-bottom: var(--ds-space-1);
  }

  .auth-ticker-name {
    flex: 1;
    color: var(--ds-color-on-dark);
  }

  .auth-ticker-value {
    color: #ffffff;
    font-weight: var(--ds-weight-semibold);
  }
</style>
