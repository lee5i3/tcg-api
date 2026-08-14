<script lang="ts">
  import { Button } from "@tcg/design-system";
  import { page } from "$app/state";
  import { appUrl } from "$lib/config";
</script>

<svelte:head>
  <title>Page not found — TCG Catalog</title>
</svelte:head>

<section class="cta-band">
  <div class="wrap">
    <h1 class="section-title">
      {page.status === 404 ? "This page isn't in the binder." : `Error ${page.status}`}
    </h1>
    <p>The page you're looking for doesn't exist.</p>
    <p class="error-actions">
      <Button variant="primary" href="/">Back to the front page</Button>
      <Button variant="inverse" href={appUrl()}>Open the app</Button>
    </p>
  </div>
</section>
