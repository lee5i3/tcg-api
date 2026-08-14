<script lang="ts">
  import { api, type CardSummary } from "$lib/api";
  import { createQuery } from "$lib/query.svelte";
  import CardGrid from "$lib/components/CardGrid.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();

  const cards = createQuery<CardSummary[]>(() => api.fetchSetCards(data.game, data.set));
</script>

<svelte:head>
  <title>{data.set} cards — TCG Catalog</title>
</svelte:head>

<section>
  <nav class="breadcrumbs">
    <a href="/">Games</a> /
    <a href={`/g/${encodeURIComponent(data.game)}`}>{data.game}</a> /
    <span>{data.set}</span>
  </nav>
  <h1>Cards</h1>
  {#if cards.state.status === "loading"}
    <p class="status">Loading…</p>
  {:else if cards.state.status === "error"}
    <p class="status status-error">Something went wrong: {cards.state.message}</p>
  {:else}
    <CardGrid game={data.game} cards={cards.state.data} />
  {/if}
</section>
