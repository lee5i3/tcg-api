<script lang="ts">
  import { Badge, Button } from "@tcg/design-system";
  import { appUrl } from "$lib/config";
  import { pricingTiers } from "$lib/pricing";

  // TODO(signup): registration/payment is not built yet — every tier CTA
  // points at the app. Swap these hrefs for the signup/checkout flow
  // (per-tier) once it exists.
  const href = appUrl();
</script>

<div class="pricing-grid">
  {#each pricingTiers as tier (tier.id)}
    <article class="pricing-card" class:pricing-card-highlighted={tier.highlighted}>
      {#if tier.highlighted}
        <p class="pricing-flag"><Badge>Most popular</Badge></p>
      {/if}
      <h3 class="pricing-name">{tier.name}</h3>
      <p class="pricing-price">
        <span class="pricing-amount">{tier.price}</span>
        <span class="pricing-period">{tier.period}</span>
      </p>
      <p class="pricing-tagline">{tier.tagline}</p>
      <ul class="pricing-features ds-bullets">
        {#each tier.features as feature (feature)}
          <li>{feature}</li>
        {/each}
      </ul>
      <Button variant={tier.highlighted ? "primary" : "secondary"} {href}>
        {tier.cta}
      </Button>
    </article>
  {/each}
</div>
<p class="pricing-note">
  Paid plans are placeholders while we're in beta — everything currently in the
  app is free to use.
</p>
