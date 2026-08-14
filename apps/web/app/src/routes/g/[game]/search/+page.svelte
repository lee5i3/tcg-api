<script lang="ts">
  import { goto } from "$app/navigation";
  import { api, type CardSummary } from "$lib/api";
  import { createQuery } from "$lib/query.svelte";
  import CardGrid from "$lib/components/CardGrid.svelte";
  import SearchBox from "$lib/components/SearchBox.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();

  const results = createQuery<CardSummary[]>(() =>
    data.q ? api.searchCards(data.game, data.q) : Promise.resolve([]),
  );

  function onSearch(value: string) {
    const base = `/g/${encodeURIComponent(data.game)}/search`;
    void goto(value ? `${base}?q=${encodeURIComponent(value)}` : base);
  }
</script>

<svelte:head>
  <title>Card search — TCG Catalog</title>
</svelte:head>

<section>
  <nav class="breadcrumbs">
    <a href="/">Games</a> /
    <a href={`/g/${encodeURIComponent(data.game)}`}>{data.game}</a> /
    <span>search</span>
  </nav>
  <h1>Card search</h1>
  <div class="toolbar">
    {#key data.q}
      <SearchBox initialValue={data.q} placeholder="Search cards by name" {onSearch} />
    {/key}
  </div>
  {#if data.q === ""}
    <p class="status">Type a card name to search.</p>
  {:else if results.state.status === "loading"}
    <p class="status">Loading…</p>
  {:else if results.state.status === "error"}
    <p class="status status-error">Something went wrong: {results.state.message}</p>
  {:else}
    <CardGrid game={data.game} cards={results.state.data} />
  {/if}
</section>
