<script lang="ts">
  import { goto } from "$app/navigation";
  import { api, type SetSummary } from "$lib/api";
  import { createQuery } from "$lib/query.svelte";
  import SearchBox from "$lib/components/SearchBox.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();

  const sets = createQuery<SetSummary[]>(() => api.fetchSets(data.game, data.query));

  function onSearch(value: string) {
    const base = `/g/${encodeURIComponent(data.game)}`;
    void goto(value ? `${base}?query=${encodeURIComponent(value)}` : base);
  }
</script>

<svelte:head>
  <title>{data.game} sets — TCG Catalog</title>
</svelte:head>

<section>
  <nav class="breadcrumbs">
    <a href="/">Games</a> / <span>{data.game}</span>
  </nav>
  <h1>Sets</h1>
  <div class="toolbar">
    {#key data.query}
      <SearchBox initialValue={data.query} placeholder="Search sets" {onSearch} />
    {/key}
    <a class="side-link" href={`/g/${encodeURIComponent(data.game)}/search`}>
      Search cards instead
    </a>
  </div>
  {#if sets.state.status === "loading"}
    <p class="status">Loading…</p>
  {:else if sets.state.status === "error"}
    <p class="status status-error">Something went wrong: {sets.state.message}</p>
  {:else if sets.state.data.length === 0}
    <p class="status">No sets found.</p>
  {:else}
    <ul class="tile-list">
      {#each sets.state.data as set (set.id)}
        <li>
          <a
            class="tile"
            href={`/g/${encodeURIComponent(data.game)}/s/${encodeURIComponent(set.key)}`}
          >
            {#if set.logoUrl}
              <img class="tile-logo" src={set.logoUrl} alt="" loading="lazy" />
            {/if}
            <span class="tile-title">{set.name}</span>
            <span class="tile-meta">
              {set.cardCount} cards{set.releaseDate ? ` · ${set.releaseDate}` : ""}
            </span>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</section>
