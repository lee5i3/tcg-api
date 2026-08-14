<script lang="ts">
  // Tasteful screenshot *placeholder*: an abstract browser-frame mockup of
  // the app, drawn inline — deliberately not a fabricated real screenshot.
  // Hex values mirror the design-system tokens.
  let { variant }: { variant: "set" | "card" | "search" } = $props();
</script>

<svg class="app-frame" viewBox="0 0 360 240" role="img" aria-label="Abstract mockup of the catalog app">
  <!-- Browser chrome on a white surface -->
  <rect x="1" y="1" width="358" height="238" rx="12" fill="#ffffff" stroke="#e2e8f0" />
  <rect x="1" y="1" width="358" height="28" rx="12" fill="#f1f5f9" />
  <rect x="1" y="15" width="358" height="14" fill="#f1f5f9" />
  <circle cx="18" cy="15" r="4" fill="#cbd5e1" />
  <circle cx="32" cy="15" r="4" fill="#cbd5e1" />
  <circle cx="46" cy="15" r="4" fill="#cbd5e1" />
  <rect x="64" y="8" width="200" height="14" rx="7" fill="#ffffff" stroke="#e2e8f0" />

  {#if variant === "set"}
    <!-- Card grid with price bars -->
    {#each [0, 1, 2, 3] as col (col)}
      {#each [0, 1] as row (row)}
        <g transform={`translate(${22 + col * 82}, ${44 + row * 94})`}>
          <rect width="66" height="58" rx="8" fill="#4f46e5" opacity={0.3 + ((col + row) % 3) * 0.22} />
          <rect y="64" width="44" height="7" rx="3.5" fill="#cbd5e1" />
          <rect y="76" width="28" height="7" rx="3.5" fill="#4f46e5" />
        </g>
      {/each}
    {/each}
  {:else if variant === "card"}
    <!-- Card detail: image left, variant price table right -->
    <rect x="22" y="44" width="120" height="168" rx="10" fill="#4f46e5" opacity="0.85" />
    <rect x="160" y="48" width="140" height="14" rx="7" fill="#0f172a" />
    <rect x="160" y="72" width="90" height="9" rx="4.5" fill="#cbd5e1" />
    {#each [0, 1, 2, 3] as row (row)}
      <g transform={`translate(160, ${96 + row * 28})`}>
        <rect width="100" height="9" rx="4.5" fill="#e2e8f0" />
        <rect x="134" width="44" height="9" rx="4.5" fill="#4f46e5" opacity={1 - row * 0.18} />
      </g>
    {/each}
  {:else}
    <!-- Search: query box and result rows -->
    <rect x="22" y="44" width="230" height="22" rx="11" fill="#ffffff" stroke="#4f46e5" />
    <rect x="260" y="44" width="52" height="22" rx="11" fill="#4f46e5" />
    {#each [0, 1, 2, 3, 4] as row (row)}
      <g transform={`translate(22, ${84 + row * 30})`}>
        <rect width="22" height="22" rx="6" fill="#4f46e5" opacity={0.35 + (row % 3) * 0.22} />
        <rect x="32" y="3" width={150 - row * 14} height="8" rx="4" fill="#cbd5e1" />
        <rect x="270" y="3" width="46" height="8" rx="4" fill="#4f46e5" />
      </g>
    {/each}
  {/if}
</svg>

<style>
  .app-frame {
    display: block;
    width: 100%;
    height: auto;
    border-radius: 16px;
    box-shadow: var(--ds-shadow-md);
  }
</style>
