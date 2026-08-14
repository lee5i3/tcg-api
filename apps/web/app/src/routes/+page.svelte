<script lang="ts">
  import { api, type Game } from "$lib/api";
  import { createQuery } from "$lib/query.svelte";

  const games = createQuery<Game[]>(() => api.fetchGames());
</script>

<svelte:head>
  <title>Games — TCG Catalog</title>
</svelte:head>

<section>
  <h1>Games</h1>
  {#if games.state.status === "loading"}
    <p class="status">Loading…</p>
  {:else if games.state.status === "error"}
    <p class="status status-error">Something went wrong: {games.state.message}</p>
  {:else if games.state.data.length === 0}
    <p class="status">No games available.</p>
  {:else}
    <ul class="tile-list">
      {#each games.state.data as game (game.id)}
        <li>
          <a class="tile" href={`/g/${encodeURIComponent(game.key)}`}>
            <span class="tile-title">{game.label}</span>
            <span class="tile-meta">{game.language}</span>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</section>
