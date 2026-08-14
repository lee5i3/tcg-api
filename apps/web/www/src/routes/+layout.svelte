<script lang="ts">
  import "@tcg/design-system/tokens.css";
  import "@tcg/design-system/base.css";
  import "../app.css";
  import { Button, SiteHeader } from "@tcg/design-system";
  import { appUrl } from "$lib/config";
  import type { LayoutProps } from "./$types";

  let { children }: LayoutProps = $props();

  const app = appUrl();
</script>

<SiteHeader tone="dark">
  <a href="/#features">Features</a>
  <a href="/#pricing">Pricing</a>
  <a href="/#faq">FAQ</a>
  <Button variant="primary" size="sm" href={app}>Open the app</Button>
</SiteHeader>

{@render children()}
