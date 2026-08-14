<script lang="ts">
  import { Alert, Button, TextField } from "@tcg/design-system";
  import { goto } from "$app/navigation";
  import { auth } from "$lib/auth.svelte";
  import AuthShell from "$lib/components/AuthShell.svelte";

  let value = $state("");
  let error = $state<string | null>(null);
  let busy = $state(false);

  async function onSubmit(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    error = null;
    const message = await auth.login(value);
    busy = false;
    if (message) {
      error = message;
    } else {
      await goto("/", { replaceState: true });
    }
  }
</script>

<svelte:head>
  <title>Login — TCG Catalog Admin</title>
</svelte:head>

<AuthShell>
  <h1 class="auth-title">Sign in to the admin</h1>
  <p class="auth-subtitle">
    Paste the catalog admin token. It is checked against the API and kept only
    for this browser session.
  </p>
  {#if error}<Alert variant="error">{error}</Alert>{/if}
  <form class="auth-form" onsubmit={onSubmit} aria-label="Admin login">
    <TextField
      tone="dark"
      label="Admin token"
      id="admin-token"
      type="password"
      autocomplete="off"
      placeholder="Paste your admin token"
      bind:value
    >
      {#snippet icon()}
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <rect x="4" y="10" width="16" height="10" rx="2" />
          <path d="M8 10V7a4 4 0 0 1 8 0v3" />
        </svg>
      {/snippet}
    </TextField>
    <Button class="auth-btn" type="submit" disabled={busy}>
      {busy ? "Checking…" : "Log in"}
    </Button>
  </form>
</AuthShell>
