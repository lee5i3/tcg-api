<script lang="ts">
  import { Alert, Button, TextField } from "@tcg/design-system";
  import { untrack } from "svelte";
  import type { CardInput, CardSummary, SetSummary } from "@tcg/api-client";
  import { auth } from "$lib/auth.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();

  interface CardFormState {
    name: string;
    number: string;
    rarity: string;
    language: string;
    tcgplayerId: string;
    imageSmall: string;
    imageLarge: string;
  }

  const emptyForm: CardFormState = {
    name: "",
    number: "",
    rarity: "",
    language: "",
    tcgplayerId: "",
    imageSmall: "",
    imageLarge: "",
  };

  function toInput(setId: string, form: CardFormState): CardInput {
    return {
      setId,
      name: form.name.trim(),
      ...(form.number.trim() ? { number: form.number.trim() } : {}),
      ...(form.rarity.trim() ? { rarity: form.rarity.trim() } : {}),
      ...(form.language.trim() ? { language: form.language.trim() } : {}),
      ...(form.tcgplayerId.trim() ? { tcgplayerId: Number(form.tcgplayerId) } : {}),
      ...(form.imageSmall.trim() ? { imageSmall: form.imageSmall.trim() } : {}),
      ...(form.imageLarge.trim() ? { imageLarge: form.imageLarge.trim() } : {}),
    };
  }

  let setInfo = $state<SetSummary | null>(null);
  let cards = $state<CardSummary[] | null>(null);
  let loadError = $state<string | null>(null);
  let formError = $state<string | null>(null);
  let notice = $state<string | null>(null);
  let busy = $state(false);
  let editing = $state<CardSummary | null>(null);
  let form = $state<CardFormState>({ ...emptyForm });

  async function load() {
    loadError = null;
    try {
      const [allSets, cardRows] = await Promise.all([
        auth.client.fetchSets(data.game),
        auth.client.fetchSetCards(data.game, data.set),
      ]);
      setInfo = allSets.find((row) => row.key === data.set || row.id === data.set) ?? null;
      cards = cardRows;
    } catch (error) {
      loadError = error instanceof Error ? error.message : "Failed to load cards.";
    }
  }

  $effect(() => {
    void data.game;
    void data.set;
    untrack(() => {
      void load();
    });
  });

  function startEdit(target: CardSummary) {
    editing = target;
    formError = null;
    form = {
      name: target.name,
      number: target.number ?? "",
      rarity: target.rarity ?? "",
      language: target.language,
      tcgplayerId: target.tcgplayerId === null ? "" : String(target.tcgplayerId),
      imageSmall: target.image ?? "",
      imageLarge: target.imageLarge ?? "",
    };
  }

  function cancelEdit() {
    editing = null;
    formError = null;
    form = { ...emptyForm };
  }

  async function onSubmit(event: SubmitEvent) {
    event.preventDefault();
    if (!setInfo) {
      formError = "Set metadata is still loading; try again in a moment.";
      return;
    }
    busy = true;
    formError = null;
    notice = null;
    try {
      const name = form.name.trim();
      if (editing) {
        await auth.client.updateCard(data.game, editing.id, toInput(setInfo.id, form));
        notice = `Updated card “${name}”.`;
      } else {
        await auth.client.createCard(data.game, toInput(setInfo.id, form));
        notice = `Created card “${name}”.`;
      }
      cancelEdit();
      await load();
    } catch (error) {
      formError = error instanceof Error ? error.message : "Failed to save card.";
    } finally {
      busy = false;
    }
  }

  async function onDelete(target: CardSummary) {
    const confirmed = window.confirm(`Delete card “${target.name}”?`);
    if (!confirmed) {
      return;
    }
    notice = null;
    loadError = null;
    try {
      await auth.client.deleteCard(data.game, target.id);
      notice = `Deleted card “${target.name}”.`;
      if (editing?.id === target.id) {
        cancelEdit();
      }
      await load();
    } catch (error) {
      loadError = error instanceof Error ? error.message : "Failed to delete card.";
    }
  }
</script>

<svelte:head>
  <title>Cards: {data.set} — TCG Catalog Admin</title>
</svelte:head>

<section>
  <p class="breadcrumbs">
    <a href="/">Games</a> /
    <a href={`/g/${encodeURIComponent(data.game)}`}>{data.game}</a> /
    {setInfo?.name ?? data.set}
  </p>
  <h1>Cards — {setInfo?.name ?? data.set}</h1>

  <h2>{editing ? `Edit card: ${editing.name}` : "New card"}</h2>
  <form class="form" onsubmit={onSubmit} aria-label={editing ? "Edit card" : "Create card"}>
    {#if formError}<Alert variant="error">{formError}</Alert>{/if}
    <div class="form-row">
      <TextField label="Name" id="card-name" bind:value={form.name} required />
      <TextField label="Number" hint="(optional)" id="card-number" bind:value={form.number} />
      <TextField label="Rarity" hint="(optional)" id="card-rarity" bind:value={form.rarity} />
    </div>
    <div class="form-row">
      <TextField
        label="Language"
        hint="(optional)"
        id="card-language"
        bind:value={form.language}
        placeholder="en"
      />
      <TextField
        label="TCGplayer ID"
        hint="(optional)"
        id="card-tcgplayer-id"
        type="number"
        min="0"
        bind:value={form.tcgplayerId}
      />
    </div>
    <div class="form-row">
      <TextField
        label="Image URL"
        hint="(optional)"
        id="card-image-small"
        type="url"
        bind:value={form.imageSmall}
      />
      <TextField
        label="Large image URL"
        hint="(optional)"
        id="card-image-large"
        type="url"
        bind:value={form.imageLarge}
      />
    </div>
    <div class="form-actions">
      <Button type="submit" disabled={busy}>
        {busy ? "Saving…" : editing ? "Save changes" : "Create card"}
      </Button>
      {#if editing}
        <Button variant="ghost" type="button" onclick={cancelEdit}>Cancel</Button>
      {/if}
    </div>
  </form>

  <h2>Existing cards</h2>
  {#if notice}<Alert variant="success">{notice}</Alert>{/if}
  {#if loadError}
    <p class="status status-error">{loadError}</p>
  {:else if cards === null}
    <p class="status">Loading…</p>
  {:else if cards.length === 0}
    <p class="status">No cards in this set yet.</p>
  {:else}
    <table class="admin-table">
      <thead>
        <tr>
          <th>Number</th>
          <th>Name</th>
          <th>Rarity</th>
          <th>TCGplayer ID</th>
          <th aria-label="Actions"></th>
        </tr>
      </thead>
      <tbody>
        {#each cards as row (row.id)}
          <tr>
            <td>{row.number || "—"}</td>
            <td>{row.name}</td>
            <td>{row.rarity ?? "—"}</td>
            <td>{row.tcgplayerId ?? "—"}</td>
            <td>
              <div class="row-actions">
                <Button variant="ghost" size="sm" type="button" onclick={() => startEdit(row)}>
                  Edit
                </Button>
                <Button variant="danger" size="sm" type="button" onclick={() => void onDelete(row)}>
                  Delete
                </Button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</section>
