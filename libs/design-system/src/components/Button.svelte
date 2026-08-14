<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLAnchorAttributes, HTMLButtonAttributes } from "svelte/elements";

  type ButtonVariant = "primary" | "secondary" | "danger" | "ghost" | "inverse" | "paper";
  type ButtonSize = "sm" | "md";

  interface BaseProps {
    variant?: ButtonVariant;
    size?: ButtonSize;
    children: Snippet;
  }

  type AnchorProps = BaseProps & HTMLAnchorAttributes & { href: string };
  type NativeProps = BaseProps & HTMLButtonAttributes & { href?: undefined };
  type Props = AnchorProps | NativeProps;

  let {
    variant = "primary",
    size = "md",
    href,
    class: className = "",
    children,
    ...rest
  }: Props = $props();

  const anchorAttrs = $derived(rest as HTMLAnchorAttributes);
  const buttonAttrs = $derived(rest as HTMLButtonAttributes);
</script>

{#if href !== undefined}
  <a
    {href}
    {...anchorAttrs}
    class="ds-btn {className}"
    class:ds-btn-primary={variant === "primary"}
    class:ds-btn-secondary={variant === "secondary"}
    class:ds-btn-danger={variant === "danger"}
    class:ds-btn-ghost={variant === "ghost"}
    class:ds-btn-inverse={variant === "inverse"}
    class:ds-btn-paper={variant === "paper"}
    class:ds-btn-sm={size === "sm"}>{@render children()}</a
  >
{:else}
  <button
    {...buttonAttrs}
    class="ds-btn {className}"
    class:ds-btn-primary={variant === "primary"}
    class:ds-btn-secondary={variant === "secondary"}
    class:ds-btn-danger={variant === "danger"}
    class:ds-btn-ghost={variant === "ghost"}
    class:ds-btn-inverse={variant === "inverse"}
    class:ds-btn-paper={variant === "paper"}
    class:ds-btn-sm={size === "sm"}>{@render children()}</button
  >
{/if}

<style>
  .ds-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--ds-space-2);
    padding: 0.6rem 1.25rem;
    border: 1px solid transparent;
    border-radius: var(--ds-radius-md);
    font-family: var(--ds-font-body);
    font-size: var(--ds-text-md);
    font-weight: var(--ds-weight-semibold);
    text-decoration: none;
    text-align: center;
    line-height: var(--ds-leading-snug);
    cursor: pointer;
    transition:
      background-color 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease,
      box-shadow 0.15s ease;
  }

  a.ds-btn:hover {
    text-decoration: none;
  }

  .ds-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .ds-btn-sm {
    padding: 0.35rem 0.85rem;
    font-size: var(--ds-text-sm);
  }

  /* Solid accent: the one call to action. */
  .ds-btn-primary {
    background: var(--ds-color-accent);
    color: var(--ds-color-on-accent);
    box-shadow: var(--ds-shadow-xs);
  }

  .ds-btn-primary:hover:not(:disabled) {
    background: var(--ds-color-accent-strong);
    color: var(--ds-color-on-accent);
    box-shadow: var(--ds-shadow-sm);
  }

  /* Quiet outline on white surfaces. */
  .ds-btn-secondary {
    background: var(--ds-color-surface);
    color: var(--ds-color-ink);
    border-color: var(--ds-color-border-strong);
    box-shadow: var(--ds-shadow-xs);
  }

  .ds-btn-secondary:hover:not(:disabled) {
    border-color: var(--ds-color-accent);
    color: var(--ds-color-accent-strong);
  }

  .ds-btn-danger {
    background: var(--ds-color-danger);
    color: #ffffff;
  }

  .ds-btn-danger:hover:not(:disabled) {
    background: var(--ds-color-danger-text);
  }

  /* Borderless, for rows of quiet actions. */
  .ds-btn-ghost {
    background: transparent;
    color: var(--ds-color-muted);
  }

  .ds-btn-ghost:hover:not(:disabled) {
    background: var(--ds-color-surface-sunken);
    color: var(--ds-color-ink);
  }

  /* Outline for dark panels (hero, CTA bands, dark header). */
  .ds-btn-inverse {
    background: transparent;
    color: var(--ds-color-on-dark);
    border-color: var(--ds-color-dark-line);
  }

  .ds-btn-inverse:hover:not(:disabled) {
    background: rgba(148, 163, 184, 0.14);
    border-color: var(--ds-color-on-dark-muted);
    color: #ffffff;
  }

  /* Filled white CTA for dark panels. */
  .ds-btn-paper {
    background: var(--ds-color-surface);
    color: var(--ds-color-ink);
  }

  .ds-btn-paper:hover:not(:disabled) {
    background: var(--ds-color-surface-sunken);
    color: var(--ds-color-ink);
  }
</style>
