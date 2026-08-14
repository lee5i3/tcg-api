<script lang="ts">
  import { Alert, Button, TextField } from "@tcg/design-system";
  import { untrack } from "svelte";
  import type { SetInput, SetSummary } from "@tcg/api-client";
  import { auth } from "$lib/auth.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();

  interface SetFormState {
    key: string;
    name: string;
    language: string;
    releaseDate: string;
    cardTotal: string;
    logoUrl: string;
  }

  const emptyForm: SetFormState = {
    key: "",
    name: "",
    language: "",
    releaseDate: "",
    cardTotal: "",
    logoUrl: "",
  };

  function toInput(form: SetFormState): SetInput {
    return {
      key: form.key.trim(),
      name: form.name.trim(),
      ...(form.language.trim() ? { language: form.language.trim() } : {}),
      ...(form.releaseDate.trim() ? { releaseDate: form.releaseDate.trim() } : {}),
      ...(form.cardTotal.trim() ? { cardTotal: Number(form.cardTotal) } : {}),
      ...(form.logoUrl.trim() ? { logoUrl: form.logoUrl.trim() } : {}),
    };
  }

  let sets = $state<SetSummary[] | null>(null);
  let search = $state("");
  let loadError = $state<string | null>(null);
  let formError = $state<string | null>(null);
  let notice = $state<string | null>(null);
  let busy = $state(false);
  let editing = $state<SetSummary | null>(null);
  let form = $state<SetFormState>({ ...emptyForm });

  async function load(query = "") {
    loadError = null;
    try {
      sets = await auth.client.fetchSets(data.game, query);
    } catch (error) {
      loadError = error instanceof Error ? error.message : "Failed to load sets.";
    }
  }

  $effect(() => {
    void data.game; // reload when navigating between games
    untrack(() => {
      void load(search);
    });
  });

  function startEdit(target: SetSummary) {
    editing = target;
    formError = null;
    form = {
      key: target.key,
      name: target.name,
      language: target.language,
      releaseDate: target.releaseDate ?? "",
      cardTotal: target.cardTotal === null ? "" : String(target.cardTotal),
      logoUrl: target.logoUrl ?? "",
    };
  }

  function cancelEdit() {
    editing = null;
    formError = null;
    form = { ...emptyForm };
  }

  async function onSubmit(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    formError = null;
    notice = null;
    try {
      const name = form.name.trim();
      if (editing) {
        await auth.client.updateSet(data.game, editing.id, toInput(form));
        notice = `Updated set “${name}”.`;
      } else {
        await auth.client.createSet(data.game, toInput(form));
        notice = `Created set “${name}”.`;
      }
      cancelEdit();
      await load(search);
    } catch (error) {
      formError = error instanceof Error ? error.message : "Failed to save set.";
    } finally {
      busy = false;
    }
  }

  async function onDelete(target: SetSummary) {
    const confirmed = window.confirm(
      `Delete set “${target.name}”? All of its cards will be deleted too.`,
    );
    if (!confirmed) {
      return;
    }
    notice = null;
    loadError = null;
    try {
      const cardsDeleted = await auth.client.deleteSet(data.game, target.id);
      notice = `Deleted set “${target.name}” (${cardsDeleted} cards removed).`;
      if (editing?.id === target.id) {
        cancelEdit();
      }
      await load(search);
    } catch (error) {
      loadError = error instanceof Error ? error.message : "Failed to delete set.";
    }
  }

  function onSearch(event: SubmitEvent) {
    event.preventDefault();
    void load(search);
  }
</script>

<svelte:head>
  <title>Sets: {data.game} — TCG Catalog Admin</title>
</svelte:head>

<section>
  <p class="breadcrumbs">
    <a href="/">Games</a> / {data.game}
  </p>
  <h1>Sets — {data.game}</h1>

  <h2>{editing ? `Edit set: ${editing.name}` : "New set"}</h2>
  <form class="form" onsubmit={onSubmit} aria-label={editing ? "Edit set" : "Create set"}>
    {#if formError}<Alert variant="error">{formError}</Alert>{/if}
    <div class="form-row">
      <TextField label="Key" id="set-key" bind:value={form.key} required disabled={editing !== null} />
      <TextField label="Name" id="set-name" bind:value={form.name} required />
      <TextField
        label="Language"
        hint="(optional)"
        id="set-language"
        bind:value={form.language}
        placeholder="en"
        disabled={editing !== null}
      />
    </div>
    <div class="form-row">
      <TextField
        label="Release date"
        hint="(optional)"
        id="set-release-date"
        type="date"
        bind:value={form.releaseDate}
      />
      <TextField
        label="Card total"
        hint="(optional)"
        id="set-card-total"
        type="number"
        min="0"
        bind:value={form.cardTotal}
      />
      <TextField label="Logo URL" hint="(optional)" id="set-logo-url" type="url" bind:value={form.logoUrl} />
    </div>
    <div class="form-actions">
      <Button type="submit" disabled={busy}>
        {busy ? "Saving…" : editing ? "Save changes" : "Create set"}
      </Button>
      {#if editing}
        <Button variant="ghost" type="button" onclick={cancelEdit}>Cancel</Button>
      {/if}
    </div>
  </form>

  <h2>Existing sets</h2>
  <form class="toolbar" onsubmit={onSearch} role="search">
    <input aria-label="Search sets" placeholder="Search sets…" bind:value={search} />
    <Button variant="ghost" type="submit">Search</Button>
  </form>
  {#if notice}<Alert variant="success">{notice}</Alert>{/if}
  {#if loadError}
    <p class="status status-error">{loadError}</p>
  {:else if sets === null}
    <p class="status">Loading…</p>
  {:else if sets.length === 0}
    <p class="status">No sets found.</p>
  {:else}
    <table class="admin-table">
      <thead>
        <tr>
          <th>Key</th>
          <th>Name</th>
          <th>Cards</th>
          <th>Release date</th>
          <th aria-label="Actions"></th>
        </tr>
      </thead>
      <tbody>
        {#each sets as row (row.id)}
          <tr>
            <td>{row.key}</td>
            <td>{row.name}</td>
            <td>{row.cardCount}{row.cardTotal !== null ? ` / ${row.cardTotal}` : ""}</td>
            <td>{row.releaseDate ?? "—"}</td>
            <td>
              <div class="row-actions">
                <Button
                  variant="ghost"
                  size="sm"
                  href={`/g/${encodeURIComponent(data.game)}/s/${encodeURIComponent(row.key)}`}
                >
                  Cards
                </Button>
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
