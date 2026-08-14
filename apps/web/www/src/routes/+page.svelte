<script lang="ts">
  import { Button } from "@tcg/design-system";
  import { appUrl } from "$lib/config";
  import { faqEntries } from "$lib/faq";
  import AppFrame from "$lib/components/AppFrame.svelte";
  import HoloCard from "$lib/components/HoloCard.svelte";
  import Pricing from "$lib/components/Pricing.svelte";

  const app = appUrl();
</script>

<svelte:head>
  <title>TCG Catalog — know what your collection is worth</title>
  <meta
    name="description"
    content="A multi-game trading card catalog with live prices. Browse every set and printing, search across games, and track your TCG collection's value."
  />
</svelte:head>

<!-- Hero: deep dark panel with a gradient accent glow -->
<header class="hero">
  <div class="wrap">
    <div class="hero-body">
      <div>
        <p class="ds-eyebrow hero-eyebrow">Multi-game card catalog · live prices</p>
        <h1>
          Know what your <span class="hero-highlight">binder</span> is really worth.
        </h1>
        <p class="hero-lede">
          TCG Catalog tracks every printing across your trading card games —
          with the lowest live market price for every variant. Browse sets,
          look up any card, and watch values move.
        </p>
        <div class="hero-ctas">
          <Button variant="primary" href={app}>Get started</Button>
          <Button variant="inverse" href="#pricing">See pricing</Button>
        </div>
        <ul class="hero-facts">
          <li>
            <span class="fact-figure">1 catalog</span>
            <span class="fact-label">games, sets, and cards</span>
          </li>
          <li>
            <span class="fact-figure">per-variant</span>
            <span class="fact-label">holo, reverse, 1st edition</span>
          </li>
          <li>
            <span class="fact-figure">live low</span>
            <span class="fact-label">market price on every card</span>
          </li>
        </ul>
      </div>
      <div class="hero-card">
        <HoloCard />
      </div>
    </div>
  </div>
</header>

<main>
  <!-- Features -->
  <section class="section" id="features">
    <div class="wrap">
      <p class="ds-eyebrow">What you get</p>
      <h2 class="section-title">A price guide that reads like a set list.</h2>
      <p class="section-lede">
        The catalog is organized the way collectors think: game, set, card,
        printing — with a price on every step of the way down.
      </p>
      <div class="feature-grid">
        <article class="feature">
          <div class="feature-swatch" aria-hidden="true"></div>
          <h3>Every game in one place</h3>
          <p>
            One consistent catalog across trading card games — no juggling a
            different price site per game.
          </p>
        </article>
        <article class="feature">
          <div class="feature-swatch" aria-hidden="true"></div>
          <h3>Prices per printing</h3>
          <p>
            Holo, reverse holo, first edition: each variant carries its own
            live lowest market price, side by side.
          </p>
        </article>
        <article class="feature">
          <div class="feature-swatch" aria-hidden="true"></div>
          <h3>Set-by-set browsing</h3>
          <p>
            Complete set lists with card counts, release dates, and a card
            grid that shows values at a glance.
          </p>
        </article>
        <article class="feature">
          <div class="feature-swatch" aria-hidden="true"></div>
          <h3>Search that knows cards</h3>
          <p>
            Type a card name and search a whole game at once — every printing
            of it, priced.
          </p>
        </article>
      </div>
    </div>
  </section>

  <!-- Screenshots (abstract mockups, not fabricated screenshots) -->
  <section class="section section-alt" id="tour">
    <div class="wrap">
      <p class="ds-eyebrow">A quick look</p>
      <h2 class="section-title">From set list to price, in three screens.</h2>
      <div class="shots-grid">
        <figure class="shot">
          <AppFrame variant="set" />
          <figcaption>
            <strong>Browse a set</strong>
            Card grid with image, number, rarity, and the lowest variant price.
          </figcaption>
        </figure>
        <figure class="shot">
          <AppFrame variant="card" />
          <figcaption>
            <strong>Open a card</strong>
            Full detail with a price table for every printing and variant.
          </figcaption>
        </figure>
        <figure class="shot">
          <AppFrame variant="search" />
          <figcaption>
            <strong>Search a game</strong>
            Find any card by name across a whole game in one query.
          </figcaption>
        </figure>
      </div>
    </div>
  </section>

  <!-- Pricing -->
  <section class="section" id="pricing">
    <div class="wrap">
      <p class="ds-eyebrow">Pricing</p>
      <h2 class="section-title">Free to browse. Fair to collect.</h2>
      <p class="section-lede">
        The catalog and live prices stay free. Paid tiers add tracking and
        tooling on top when you're ready for them.
      </p>
      <Pricing />
    </div>
  </section>

  <!-- FAQ -->
  <section class="section section-alt" id="faq">
    <div class="wrap">
      <p class="ds-eyebrow">FAQ</p>
      <h2 class="section-title">Fair questions.</h2>
      <div class="faq-list">
        {#each faqEntries as entry (entry.question)}
          <details class="faq-item">
            <summary>{entry.question}</summary>
            <p>{entry.answer}</p>
          </details>
        {/each}
      </div>
    </div>
  </section>

  <!-- Closing CTA -->
  <section class="cta-band">
    <div class="wrap">
      <h2 class="section-title">Your collection has a number.</h2>
      <p>Open the catalog and find out what it is.</p>
      <Button variant="primary" href={app}>Open the app</Button>
    </div>
  </section>
</main>

<footer class="footer">
  <div class="wrap">
    <span>© 2026 TCG Catalog</span>
    <nav class="footer-links" aria-label="Footer">
      <a href="#features">Features</a>
      <a href="#pricing">Pricing</a>
      <a href="#faq">FAQ</a>
      <a href={app}>Open the app</a>
    </nav>
    <span>Prices are market estimates, not offers.</span>
  </div>
</footer>
