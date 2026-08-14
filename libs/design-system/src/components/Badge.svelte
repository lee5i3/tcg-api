<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    /** "accent" is a tinted blue-violet chip; "neutral" is a quiet slate chip. */
    variant?: "accent" | "neutral";
    children: Snippet;
  }

  let { variant = "accent", children }: Props = $props();
</script>

<span
  class="ds-badge"
  class:ds-badge-accent={variant === "accent"}
  class:ds-badge-neutral={variant === "neutral"}>{@render children()}</span
>

<style>
  .ds-badge {
    display: inline-block;
    padding: 0.22em 0.75em;
    border-radius: var(--ds-radius-full);
    border: 1px solid transparent;
    font-family: var(--ds-font-body);
    font-size: var(--ds-text-xs);
    font-weight: var(--ds-weight-semibold);
    line-height: var(--ds-leading-snug);
    vertical-align: middle;
  }

  .ds-badge-accent {
    background: var(--ds-color-accent-soft);
    border-color: var(--ds-color-accent-border);
    color: var(--ds-color-accent-strong);
  }

  .ds-badge-neutral {
    background: var(--ds-color-surface-sunken);
    border-color: var(--ds-color-border);
    color: var(--ds-color-muted);
  }
</style>
