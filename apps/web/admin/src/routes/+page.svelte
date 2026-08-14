<script lang="ts">
  import { Alert, Button, TextField } from "@tcg/design-system";
  import type { Game } from "@tcg/api-client";
  import { auth } from "$lib/auth.svelte";

  let games = $state<Game[] | null>(null);
  let loadError = $state<string | null>(null);
  let formError = $state<string | null>(null);
  let busy = $state(false);
  let key = $state("");
  let label = $state("");
  let language = $state("");

  async function load() {
    loadError = null;
    try {
      games = await auth.client.fetchGames();
    } catch (error) {
      loadError = error instanceof Error ? error.message : "Failed to load games.";
    }
  }

  $effect(() => {
    void load();
  });

  async function onSubmit(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    formError = null;
    try {
      await auth.client.createGame({
        key: key.trim(),
        label: label.trim(),
        ...(language.trim() ? { language: language.trim() } : {}),
      });
      key = "";
      label = "";
      language = "";
      await load();
    } catch (error) {
      formError = error instanceof Error ? error.message : "Failed to create game.";
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head>
  <title>Games — TCG Catalog Admin</title>
</svelte:head>

<section>
  <h1>Games</h1>

  <h2>New game</h2>
  <form class="form" onsubmit={onSubmit} aria-label="Create game">
    {#if formError}<Alert variant="error">{formError}</Alert>{/if}
    <div class="form-row">
      <TextField label="Key" id="game-key" bind:value={key} required />
      <TextField label="Label" id="game-label" bind:value={label} required />
      <TextField
        label="Language"
        hint="(optional)"
        id="game-language"
        bind:value={language}
        placeholder="en"
      />
    </div>
    <div class="form-actions">
      <Button type="submit" disabled={busy}>
        {busy ? "Creating…" : "Create game"}
      </Button>
    </div>
  </form>

  <h2>Existing games</h2>
  {#if loadError}
    <p class="status status-error">{loadError}</p>
  {:else if games === null}
    <p class="status">Loading…</p>
  {:else if games.length === 0}
    <p class="status">No games yet.</p>
  {:else}
    <table class="admin-table">
      <thead>
        <tr>
          <th>Key</th>
          <th>Label</th>
          <th>Language</th>
          <th aria-label="Actions"></th>
        </tr>
      </thead>
      <tbody>
        {#each games as game (game.id)}
          <tr>
            <td>{game.key}</td>
            <td>{game.label}</td>
            <td>{game.language}</td>
            <td>
              <div class="row-actions">
                <Button variant="ghost" size="sm" href={`/g/${encodeURIComponent(game.key)}`}>
                  Manage sets
                </Button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</section>
