<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLAttributes } from "svelte/elements";

  interface Props extends HTMLAttributes<HTMLDivElement> {
    /** "md" pads the card body; "none" for cards with their own header/body blocks. */
    padding?: "none" | "md";
    children: Snippet;
  }

  let { padding = "md", class: className = "", children, ...rest }: Props = $props();
</script>

<div
  {...rest}
  class="ds-card {className}"
  class:ds-card-padded={padding === "md"}
>
  {@render children()}
</div>

<style>
  .ds-card {
    background: var(--ds-color-surface);
    border: 1px solid var(--ds-color-border);
    border-radius: var(--ds-radius-lg);
    box-shadow: var(--ds-shadow-sm);
    overflow: hidden;
  }

  .ds-card-padded {
    padding: var(--ds-space-5);
  }
</style>
