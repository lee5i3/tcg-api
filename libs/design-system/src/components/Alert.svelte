<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    variant?: "error" | "success";
    children: Snippet;
  }

  let { variant = "error", children }: Props = $props();
</script>

<p
  class="ds-alert"
  class:ds-alert-error={variant === "error"}
  class:ds-alert-success={variant === "success"}
  role="alert">{@render children()}</p
>

<style>
  .ds-alert {
    margin: 0;
    padding: var(--ds-space-3) var(--ds-space-4);
    border: 1px solid;
    border-radius: var(--ds-radius-md);
    font-size: var(--ds-text-sm);
    font-weight: var(--ds-weight-medium);
  }

  .ds-alert-error {
    background: var(--ds-color-danger-bg);
    border-color: var(--ds-color-danger-border);
    color: var(--ds-color-danger-text);
  }

  .ds-alert-success {
    background: var(--ds-color-success-bg);
    border-color: var(--ds-color-success-border);
    color: var(--ds-color-success);
  }
</style>
