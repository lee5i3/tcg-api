<script lang="ts">
  // OAuth landing pad: the backend redirects here with
  // #token=<jwt>&redirect=<path> in the URL fragment (never the query, so
  // the JWT is not sent to any server or written to logs). We store the
  // token, validate it via /v1/auth/me, then continue to the app.
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { sanitizeRedirect, setSessionFromToken } from "$lib/auth.svelte";

  const FAILURE = `/login?error=${encodeURIComponent("social sign-in failed")}`;

  onMount(async () => {
    const params = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const token = params.get("token");
    if (!token) {
      await goto(FAILURE, { replaceState: true });
      return;
    }
    try {
      await setSessionFromToken(token);
      await goto(sanitizeRedirect(params.get("redirect")), { replaceState: true });
    } catch {
      // setSessionFromToken already cleared the half-open session.
      await goto(FAILURE, { replaceState: true });
    }
  });
</script>

<svelte:head>
  <title>Signing you in… — TCG Catalog</title>
</svelte:head>

<div class="callback">
  <p class="callback-text" role="status">Signing you in…</p>
</div>

<style>
  .callback {
    min-height: 100vh;
    display: grid;
    place-items: center;
    background: var(--ds-color-dark);
  }

  .callback-text {
    margin: 0;
    color: var(--ds-color-on-dark-muted);
    font-size: var(--ds-text-md);
  }
</style>
