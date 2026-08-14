<script lang="ts">
  // Full-viewport dark split-screen for auth flows: a near-black form column
  // on the left (brand lockup top-left, content vertically centered) and an
  // optional marketing panel on the right with a soft indigo radial glow.
  // The right panel hides below 900px and the form column becomes a single
  // centered column.
  import type { Snippet } from "svelte";

  interface Props {
    /** Brand lockup pinned to the top-left of the form column. */
    brand?: Snippet;
    /** Form column content, vertically centered. */
    children: Snippet;
    /** Right marketing panel; hidden on narrow viewports. */
    aside?: Snippet;
  }

  let { brand, children, aside }: Props = $props();
</script>

<div class="ds-authsplit" class:ds-authsplit-two={aside !== undefined}>
  <div class="ds-authsplit-form">
    {#if brand}
      <header class="ds-authsplit-brand">{@render brand()}</header>
    {/if}
    <div class="ds-authsplit-main">{@render children()}</div>
  </div>
  {#if aside}
    <aside class="ds-authsplit-aside">{@render aside()}</aside>
  {/if}
</div>

<style>
  .ds-authsplit {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    min-height: 100vh;
    min-height: 100dvh;
    background: var(--ds-color-dark);
    color: var(--ds-color-on-dark);
  }

  .ds-authsplit-two {
    grid-template-columns: minmax(0, 45fr) minmax(0, 55fr);
  }

  /* The accent focus ring is too dim on #0b1120; brighten it for anything
     rendered inside the split (buttons, links, slotted content). */
  .ds-authsplit :global(:focus-visible) {
    outline-color: #a5b4fc;
  }

  .ds-authsplit-form {
    display: flex;
    flex-direction: column;
    padding: var(--ds-space-6);
  }

  .ds-authsplit-brand {
    flex: none;
  }

  .ds-authsplit-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    width: 100%;
    max-width: 24rem;
    margin: 0 auto;
    padding: var(--ds-space-6) 0;
  }

  .ds-authsplit-aside {
    position: relative;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
    padding: var(--ds-space-8) var(--ds-space-7);
    border-left: 1px solid var(--ds-color-dark-line);
    /* A slightly lifted dark shade with soft indigo/violet glows. */
    background:
      radial-gradient(80rem 55rem at 18% 12%, rgba(79, 70, 229, 0.32), transparent 62%),
      radial-gradient(60rem 45rem at 92% 90%, rgba(124, 58, 237, 0.22), transparent 60%),
      #101830;
  }

  @media (max-width: 900px) {
    .ds-authsplit-two {
      grid-template-columns: minmax(0, 1fr);
    }

    .ds-authsplit-aside {
      display: none;
    }

    .ds-authsplit-form {
      padding: var(--ds-space-5);
    }
  }
</style>
