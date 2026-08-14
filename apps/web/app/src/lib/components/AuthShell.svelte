<script lang="ts">
  // Shared chrome for /login and /register: the design-system's dark
  // split-screen with our brand lockup top-left and an identical marketing
  // panel (static "live prices" stat card) on the right.
  import { AuthSplit, BrandMark } from "@tcg/design-system";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  // Static, tasteful marketing numbers — the rows sum to the header figure.
  const rows = [
    { name: "Zapdos ex · 151", price: "$412.90", delta: "▲ 12.4%", up: true },
    { name: "Charizard · Base", price: "$399.99", delta: "▲ 2.1%", up: true },
    { name: "Blue-Eyes W. Dragon · LOB", price: "$289.00", delta: "▼ 1.8%", up: false },
    { name: "Sol Ring · Commander", price: "$182.63", delta: "▲ 0.9%", up: true },
  ];
</script>

<AuthSplit>
  {#snippet brand()}
    <div class="auth-brand">
      <a class="auth-brand-link" href="/"><BrandMark /></a>
      <p class="auth-tagline">Multi-game card catalog</p>
    </div>
  {/snippet}

  {@render children()}

  {#snippet aside()}
    <div class="auth-pitch">
      <h2 class="auth-pitch-title">Know what your binder is really worth.</h2>
      <p class="auth-pitch-copy">
        Live market prices for every game you collect — sets, singles, and
        variants in one catalog.
      </p>
      <div class="auth-ticker">
        <div class="auth-ticker-head">
          <span class="auth-ticker-dot" aria-hidden="true"></span>
          <span class="auth-ticker-label">Live market prices</span>
          <span class="auth-ticker-total ds-tabular">$1,284.52</span>
        </div>
        <ul class="auth-ticker-rows">
          {#each rows as row (row.name)}
            <li class="auth-ticker-row">
              <span class="auth-ticker-name">{row.name}</span>
              <span class="auth-ticker-price ds-tabular">{row.price}</span>
              <span
                class="auth-ticker-delta ds-tabular"
                class:auth-delta-up={row.up}
                class:auth-delta-down={!row.up}>{row.delta}</span
              >
            </li>
          {/each}
        </ul>
      </div>
    </div>
  {/snippet}
</AuthSplit>

<style>
  /* ---- Brand lockup (top-left of the form column) ------------------------ */

  .auth-brand {
    display: flex;
    flex-direction: column;
    gap: var(--ds-space-1);
  }

  .auth-brand-link {
    align-self: flex-start;
    color: var(--ds-color-on-dark);
    text-decoration: none;
    font-size: var(--ds-text-base);
  }

  .auth-brand-link:hover {
    color: #ffffff;
    text-decoration: none;
  }

  .auth-tagline {
    margin: 0;
    padding-left: 2.35em; /* clear the brand tile, align with the wordmark */
    font-size: var(--ds-text-xs);
    color: var(--ds-color-on-dark-muted);
  }

  /* ---- Right panel: marketing moment ------------------------------------- */

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

  /* Floating stat card: a mini portfolio tracker. */
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

  .auth-ticker-total {
    margin-left: auto;
    font-size: var(--ds-text-lg);
    font-weight: var(--ds-weight-bold);
    color: #ffffff;
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
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ds-color-on-dark);
  }

  .auth-ticker-price {
    color: var(--ds-color-on-dark);
    font-weight: var(--ds-weight-semibold);
  }

  .auth-ticker-delta {
    min-width: 4.25rem;
    text-align: right;
  }

  /* Brighter semantic green/red: legible on the raised dark card. */
  .auth-delta-up {
    color: #4ade80;
  }

  .auth-delta-down {
    color: #f87171;
  }
</style>
