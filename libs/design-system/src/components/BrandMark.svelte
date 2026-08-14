<script lang="ts">
  // Brand lockup: a gradient rounded-square tile holding two offset rounded
  // rectangles (a stack of cards), plus a mixed-case wordmark. The tile keeps
  // its own gradient so it reads on light and dark surfaces alike; the
  // wordmark inherits currentColor.
  interface Props {
    /** Hide the wordmark to use the tile glyph alone. */
    wordmark?: boolean;
  }

  let { wordmark = true }: Props = $props();

  // Unique gradient id: several marks can render on one page.
  const uid = $props.id();
  const gradientId = `ds-brandmark-fill-${uid}`;
</script>

<span class="ds-brandmark">
  <svg
    class="ds-brandmark-tile"
    viewBox="0 0 32 32"
    aria-hidden="true"
    focusable="false"
  >
    <defs>
      <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stop-color="#4f46e5" />
        <stop offset="1" stop-color="#7c3aed" />
      </linearGradient>
    </defs>
    <rect width="32" height="32" rx="9" fill="url(#{gradientId})" />
    <!-- Stacked cards: back card tilted and translucent, front card solid. -->
    <rect
      x="7"
      y="7"
      width="11.5"
      height="15.5"
      rx="2.4"
      fill="#ffffff"
      opacity="0.55"
      transform="rotate(-12 12.75 14.75)"
    />
    <rect x="13.5" y="9.5" width="11.5" height="15.5" rx="2.4" fill="#ffffff" />
  </svg>
  {#if wordmark}<span class="ds-brandmark-word">TCG&nbsp;Catalog</span>{/if}
</span>

<style>
  .ds-brandmark {
    display: inline-flex;
    align-items: center;
    gap: 0.55em;
    line-height: 1;
  }

  .ds-brandmark-tile {
    width: 1.75em;
    height: 1.75em;
    flex: none;
    display: block;
  }

  .ds-brandmark-word {
    font-family: var(--ds-font-body);
    font-size: 1.05em;
    font-weight: var(--ds-weight-bold);
    letter-spacing: var(--ds-tracking-tight);
    color: currentColor;
  }
</style>
