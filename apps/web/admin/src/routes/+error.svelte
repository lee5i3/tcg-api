<script lang="ts">
  import { page } from "$app/state";
</script>

<svelte:head>
  <title>Error — TCG Catalog Admin</title>
</svelte:head>

{#if page.status === 404}
  <p class="status">Page not found.</p>
{:else}
  <p class="status status-error">
    Something went wrong: {page.error?.message ?? `Error ${page.status}`}
  </p>
{/if}
