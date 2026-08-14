<script lang="ts">
  import type { Snippet } from "svelte";
  import Badge from "./Badge.svelte";
  import BrandMark from "./BrandMark.svelte";

  interface Props {
    /** Small chip after the brand, e.g. "Admin". */
    badge?: string;
    /** "light" for white/translucent pages, "dark" for marketing heroes. */
    tone?: "light" | "dark";
    /** Right-hand side: nav links, session info, actions. */
    children?: Snippet;
  }

  let { badge, tone = "light", children }: Props = $props();
</script>

<header class="ds-header" class:ds-header-dark={tone === "dark"}>
  <div class="ds-header-inner">
    <a class="ds-header-brand" href="/">
      <BrandMark />
    </a>
    {#if badge}<Badge variant="neutral">{badge}</Badge>{/if}
    {#if children}
      <div class="ds-header-actions">{@render children()}</div>
    {/if}
  </div>
</header>

<style>
  .ds-header {
    position: sticky;
    top: 0;
    z-index: 50;
    background: rgba(255, 255, 255, 0.85);
    -webkit-backdrop-filter: blur(12px) saturate(1.4);
    backdrop-filter: blur(12px) saturate(1.4);
    color: var(--ds-color-ink);
    border-bottom: 1px solid var(--ds-color-border);
  }

  .ds-header-dark {
    background: rgba(11, 17, 32, 0.8);
    color: var(--ds-color-on-dark);
    border-bottom: 1px solid var(--ds-color-dark-line);
  }

  .ds-header-inner {
    max-width: var(--ds-measure);
    margin: 0 auto;
    padding: var(--ds-space-3) var(--ds-space-5);
    display: flex;
    align-items: center;
    gap: var(--ds-space-3);
  }

  .ds-header-brand {
    display: inline-flex;
    align-items: center;
    color: inherit;
    text-decoration: none;
    font-size: var(--ds-text-base);
  }

  .ds-header-brand:hover {
    color: inherit;
    text-decoration: none;
  }

  .ds-header-actions {
    margin-left: auto;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--ds-space-4);
  }

  .ds-header-actions :global(a:not(.ds-btn)) {
    color: var(--ds-color-muted);
    text-decoration: none;
    font-size: var(--ds-text-sm);
    font-weight: var(--ds-weight-medium);
  }

  .ds-header-actions :global(a:not(.ds-btn):hover) {
    color: var(--ds-color-ink);
  }

  .ds-header-dark .ds-header-actions :global(a:not(.ds-btn)) {
    color: var(--ds-color-on-dark-muted);
  }

  .ds-header-dark .ds-header-actions :global(a:not(.ds-btn):hover) {
    color: #ffffff;
  }
</style>
