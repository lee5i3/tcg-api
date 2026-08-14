<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLInputAttributes } from "svelte/elements";

  interface Props extends Omit<HTMLInputAttributes, "id" | "value" | "class"> {
    label: string;
    /** Small de-emphasized suffix after the label, e.g. "(optional)". */
    hint?: string;
    /** Inline field error, announced via role="alert". */
    error?: string;
    /** Optional decorative leading icon rendered inside the field. */
    icon?: Snippet;
    /** "light" for white surfaces (default), "dark" for near-black panels. */
    tone?: "light" | "dark";
    id?: string;
    value?: string;
  }

  let {
    label,
    hint,
    error,
    icon,
    tone = "light",
    id,
    value = $bindable(""),
    ...rest
  }: Props = $props();

  const uid = $props.id();
  const inputId = $derived(id ?? `ds-field-${uid}`);
  const errorId = $derived(`${inputId}-error`);
</script>

<div class="ds-field" class:ds-field-dark={tone === "dark"}>
  <label class="ds-field-label" for={inputId}
    >{label}{#if hint}&nbsp;<span class="ds-field-hint">{hint}</span>{/if}</label
  >
  <div class="ds-field-shell">
    {#if icon}
      <span class="ds-field-icon" aria-hidden="true">{@render icon()}</span>
    {/if}
    <input
      class="ds-field-input"
      class:ds-field-input-iconed={icon !== undefined}
      id={inputId}
      bind:value
      aria-invalid={error ? "true" : undefined}
      aria-describedby={error ? errorId : undefined}
      {...rest}
    />
  </div>
  {#if error}<p class="ds-field-error" id={errorId} role="alert">{error}</p>{/if}
</div>

<style>
  .ds-field {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    flex: 1;
    min-width: 10rem;
  }

  .ds-field-label {
    font-size: var(--ds-text-sm);
    font-weight: var(--ds-weight-medium);
    color: var(--ds-color-ink);
  }

  .ds-field-hint {
    color: var(--ds-color-muted);
    font-weight: var(--ds-weight-normal);
  }

  .ds-field-shell {
    position: relative;
    display: flex;
  }

  .ds-field-icon {
    position: absolute;
    left: 0.85rem;
    top: 50%;
    transform: translateY(-50%);
    display: inline-flex;
    color: var(--ds-color-muted);
    pointer-events: none;
  }

  .ds-field-icon :global(svg) {
    width: 1.05rem;
    height: 1.05rem;
    display: block;
  }

  .ds-field-input {
    flex: 1;
    min-width: 0;
    padding: 0.55rem 0.85rem;
    border: 1px solid var(--ds-color-border-strong);
    border-radius: var(--ds-radius-md);
    background: var(--ds-color-surface);
    color: var(--ds-color-ink);
    font-family: var(--ds-font-body);
    font-size: var(--ds-text-md);
    font-weight: var(--ds-weight-normal);
    box-shadow: var(--ds-shadow-xs);
    transition:
      border-color 0.15s ease,
      box-shadow 0.15s ease;
  }

  .ds-field-input-iconed {
    padding-left: 2.5rem;
  }

  .ds-field-input:focus {
    outline: none;
    border-color: var(--ds-color-accent);
    box-shadow: 0 0 0 3px var(--ds-color-accent-soft);
  }

  .ds-field-input:disabled {
    background: var(--ds-color-surface-sunken);
    color: var(--ds-color-muted);
    box-shadow: none;
  }

  .ds-field-input[aria-invalid="true"] {
    border-color: var(--ds-color-danger);
  }

  .ds-field-input[aria-invalid="true"]:focus {
    box-shadow: 0 0 0 3px var(--ds-color-danger-bg);
  }

  .ds-field-error {
    margin: 0;
    font-size: var(--ds-text-sm);
    color: var(--ds-color-danger-text);
  }

  /* ---- Dark tone: fields on near-black panels --------------------------- */

  .ds-field-dark .ds-field-label {
    color: var(--ds-color-on-dark);
  }

  .ds-field-dark .ds-field-hint,
  .ds-field-dark .ds-field-icon {
    color: var(--ds-color-on-dark-muted);
  }

  .ds-field-dark .ds-field-input {
    background: var(--ds-color-dark-raised);
    border-color: var(--ds-color-dark-line);
    color: var(--ds-color-on-dark);
    box-shadow: none;
  }

  .ds-field-dark .ds-field-input::placeholder {
    color: var(--ds-color-on-dark-muted);
    opacity: 1;
  }

  /* Brighter indigo ring so focus stays obvious on #0b1120. */
  .ds-field-dark .ds-field-input:focus {
    border-color: #818cf8;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.35);
  }

  .ds-field-dark .ds-field-input:disabled {
    background: var(--ds-color-dark);
    color: var(--ds-color-on-dark-muted);
  }

  .ds-field-dark .ds-field-input[aria-invalid="true"] {
    border-color: #f87171;
  }

  .ds-field-dark .ds-field-input[aria-invalid="true"]:focus {
    box-shadow: 0 0 0 3px rgba(248, 113, 113, 0.3);
  }

  .ds-field-dark .ds-field-error {
    color: #fca5a5;
  }
</style>
