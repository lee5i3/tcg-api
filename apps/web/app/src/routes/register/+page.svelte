<script lang="ts">
  import { Alert, Button, TextField } from "@tcg/design-system";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { api, ApiError } from "$lib/api";
  import { sanitizeRedirect, setSession } from "$lib/auth.svelte";
  import AuthShell from "$lib/components/AuthShell.svelte";
  import SocialButtons from "$lib/components/SocialButtons.svelte";

  let name = $state("");
  let email = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let error = $state("");
  let submitting = $state(false);

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    error = "";
    if (password !== confirmPassword) {
      error = "Passwords do not match.";
      return;
    }
    submitting = true;
    try {
      const trimmedName = name.trim();
      const session = await api.register(
        trimmedName ? { email, password, name: trimmedName } : { email, password },
      );
      setSession(session);
      await goto(sanitizeRedirect(page.url.searchParams.get("redirect")));
    } catch (err) {
      if (err instanceof ApiError) {
        error = err.message;
      } else {
        error = "Something went wrong. Please try again.";
      }
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Create an account — TCG Catalog</title>
</svelte:head>

<AuthShell>
  <h1 class="auth-title">Create your account</h1>
  <p class="auth-subtitle">Start tracking every card you collect.</p>
  {#if error}<Alert variant="error">{error}</Alert>{/if}
  <form class="auth-form" {onsubmit}>
    <TextField
      tone="dark"
      label="Name"
      hint="(optional)"
      type="text"
      name="name"
      autocomplete="name"
      placeholder="Ash Ketchum"
      bind:value={name}
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
          <circle cx="12" cy="8" r="4" />
          <path d="M4 21a8 8 0 0 1 16 0" />
        </svg>
      {/snippet}
    </TextField>
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
      autocomplete="new-password"
      placeholder="8+ characters"
      required
      minlength={8}
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
    <TextField
      tone="dark"
      label="Confirm password"
      type="password"
      name="confirmPassword"
      autocomplete="new-password"
      placeholder="Repeat your password"
      required
      bind:value={confirmPassword}
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
      {submitting ? "Creating account…" : "Create account"}
    </Button>
  </form>
  <div class="auth-divider" aria-hidden="true"><span>or</span></div>
  <Button variant="inverse" class="auth-btn" href={`/login${page.url.search}`}>
    Sign in instead
  </Button>
  <SocialButtons redirect={page.url.searchParams.get("redirect")} />
</AuthShell>
