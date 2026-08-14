<script lang="ts">
  import { Button } from "@tcg/design-system";

  let {
    initialValue = "",
    placeholder,
    onSearch,
  }: {
    initialValue?: string;
    placeholder: string;
    onSearch: (value: string) => void;
  } = $props();

  // The initial value is captured on purpose: the parent recreates this
  // component with {#key} whenever the URL query changes.
  // svelte-ignore state_referenced_locally
  let value = $state(initialValue);

  function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    onSearch(value.trim());
  }
</script>

<form class="search-box" role="search" onsubmit={handleSubmit}>
  <input type="search" bind:value {placeholder} aria-label={placeholder} />
  <Button type="submit">Search</Button>
</form>
