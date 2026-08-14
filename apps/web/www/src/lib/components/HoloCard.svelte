<script lang="ts">
  // Signature hero graphic: a stylized trading card as a modern
  // portfolio-tracker tile — dark surface, gradient art window, green
  // price sparkline. Deliberately abstract — a mockup, not a real
  // card/screenshot. Hex values mirror tokens.css.
</script>

<svg
  class="holo-card"
  viewBox="0 0 300 420"
  role="img"
  aria-label="Stylized trading card with a rising price chart"
>
  <defs>
    <linearGradient id="card-art" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#4f46e5" />
      <stop offset="1" stop-color="#7c3aed" />
    </linearGradient>
    <linearGradient id="spark-fill" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#22c55e" stop-opacity="0.35" />
      <stop offset="1" stop-color="#22c55e" stop-opacity="0" />
    </linearGradient>
    <linearGradient id="sheen" x1="0" y1="0" x2="1" y2="0.2">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0" />
      <stop offset="0.5" stop-color="#ffffff" stop-opacity="0.18" />
      <stop offset="1" stop-color="#ffffff" stop-opacity="0" />
    </linearGradient>
    <clipPath id="card-clip">
      <rect x="10" y="10" width="280" height="400" rx="22" />
    </clipPath>
  </defs>

  <!-- Card body: raised dark surface with a subtle border -->
  <rect x="10" y="10" width="280" height="400" rx="22" fill="#131b2e" stroke="#2c3a55" />

  <!-- Name bar and rarity chip -->
  <rect x="30" y="30" width="132" height="14" rx="7" fill="#e2e8f0" opacity="0.9" />
  <rect x="234" y="30" width="36" height="14" rx="7" fill="url(#card-art)" />

  <!-- Art window: gradient accent panel -->
  <rect x="30" y="58" width="240" height="168" rx="14" fill="url(#card-art)" />
  <!-- Abstract creature silhouette -->
  <path d="M90 200c10-52 34-84 60-84s50 32 60 84z" fill="#ffffff" opacity="0.28" />
  <circle cx="150" cy="106" r="22" fill="#ffffff" opacity="0.28" />

  <!-- Price sparkline panel -->
  <rect x="30" y="244" width="240" height="104" rx="14" fill="#0b1120" stroke="#2c3a55" />
  <polygon
    points="46,326 78,318 110,322 142,300 174,306 206,284 238,288 254,268 254,336 46,336"
    fill="url(#spark-fill)"
  />
  <polyline
    points="46,326 78,318 110,322 142,300 174,306 206,284 238,288 254,268"
    fill="none"
    stroke="#22c55e"
    stroke-width="3"
    stroke-linecap="round"
    stroke-linejoin="round"
  />
  <circle cx="254" cy="268" r="4.5" fill="#22c55e" />
  <text x="46" y="274" class="holo-card-price">$412.90</text>
  <text x="46" y="290" class="holo-card-delta">▲ 12.4% this month</text>

  <!-- Variant rows -->
  <rect x="30" y="364" width="150" height="10" rx="5" fill="#334155" />
  <rect x="222" y="364" width="48" height="10" rx="5" fill="#e2e8f0" opacity="0.9" />
  <rect x="30" y="384" width="118" height="10" rx="5" fill="#334155" opacity="0.75" />
  <rect x="222" y="384" width="48" height="10" rx="5" fill="#4ade80" opacity="0.9" />

  <!-- Soft sheen across the sleeve (paused when reduced motion is preferred) -->
  <g clip-path="url(#card-clip)">
    <rect class="holo-card-sheen" x="-160" y="-40" width="120" height="520" fill="url(#sheen)" transform="rotate(14)" />
  </g>
</svg>

<style>
  .holo-card {
    display: block;
    width: 100%;
    height: auto;
    filter: drop-shadow(0 24px 48px rgba(79, 70, 229, 0.35));
  }

  .holo-card-price {
    font-family: var(--ds-font-body);
    font-variant-numeric: tabular-nums;
    font-size: 22px;
    font-weight: 700;
    fill: #ffffff;
  }

  .holo-card-delta {
    font-family: var(--ds-font-body);
    font-variant-numeric: tabular-nums;
    font-size: 11px;
    font-weight: 600;
    fill: #4ade80;
  }

  .holo-card-sheen {
    animation: sheen-sweep 5.5s ease-in-out infinite;
  }

  @keyframes sheen-sweep {
    0%,
    55% {
      transform: rotate(14deg) translateX(0);
    }
    85%,
    100% {
      transform: rotate(14deg) translateX(560px);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .holo-card-sheen {
      animation: none;
      opacity: 0;
    }
  }
</style>
