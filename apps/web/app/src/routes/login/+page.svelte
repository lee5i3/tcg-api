<script lang="ts">
  import { Alert, Button, TextField } from "@tcg/design-system";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { api, ApiError } from "$lib/api";
  import { sanitizeRedirect, setSession } from "$lib/auth.svelte";
  import AuthShell from "$lib/components/AuthShell.svelte";
  import SocialButtons from "$lib/components/SocialButtons.svelte";

  let email = $state("");
  let password = $state("");
  // Seed from ?error= so failures the backend redirects back with
  // (e.g. an aborted OAuth round-trip) surface in the same Alert.
  let error = $state(page.url.searchParams.get("error") ?? "");
  let submitting = $state(false);

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    error = "";
    submitting = true;
    try {
      const session = await api.login({ email, password });
      setSession(session);
      await goto(sanitizeRedirect(page.url.searchParams.get("redirect")));
    } catch (err) {
      if (err instanceof ApiError) {
        error = err.status === 401 ? "invalid email or password" : err.message;
      } else {
        error = "Something went wrong. Please try again.";
      }
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Sign in — TCG Catalog</title>
</svelte:head>

<AuthShell>
  <h1 class="auth-title">Sign in to your catalog</h1>
  <p class="auth-subtitle">Pick up where your collection left off.</p>
  {#if error}<Alert variant="error">{error}</Alert>{/if}
  <form class="auth-form" {onsubmit}>
    <TextField
      tone="dark"
      label="Email"
      type="email"
      name="email"
      autocomplete="email"
      placeholder="you@example.com"
      required
      bind:value={email}
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
          <rect x="3" y="5" width="18" height="14" rx="2" />
          <path d="m3 7 9 6 9-6" />
        </svg>
      {/snippet}
    </TextField>
    <TextField
      tone="dark"
      label="Password"
      type="password"
      name="password"
      autocomplete="current-password"
      placeholder="Your password"
      required
      bind:value={password}
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
          <rect x="4" y="11" width="16" height="10" rx="2" />
          <path d="M8 11V7a4 4 0 0 1 8 0v4" />
        </svg>
      {/snippet}
    </TextField>
    <Button type="submit" class="auth-btn" disabled={submitting}>
      {submitting ? "Signing in…" : "Sign in"}
    </Button>
  </form>
  <div class="auth-divider" aria-hidden="true"><span>or</span></div>
  <Button variant="inverse" class="auth-btn" href={`/register${page.url.search}`}>
    Create an account
  </Button>
  <SocialButtons redirect={page.url.searchParams.get("redirect")} />
</AuthShell>
