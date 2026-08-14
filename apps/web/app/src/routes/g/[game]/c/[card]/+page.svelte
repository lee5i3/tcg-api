<script lang="ts">
  import { api, formatPrice, type CardDetail } from "$lib/api";
  import { createQuery } from "$lib/query.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();

  const card = createQuery<CardDetail>(() => api.fetchCard(data.game, data.card));
</script>

<svelte:head>
  <title>
    {card.state.status === "ready" ? `${card.state.data.name} — TCG Catalog` : "Card — TCG Catalog"}
  </title>
</svelte:head>

<section>
  <nav class="breadcrumbs">
    <a href="/">Games</a> /
    <a href={`/g/${encodeURIComponent(data.game)}`}>{data.game}</a> /
    <span>card</span>
  </nav>
  {#if card.state.status === "loading"}
    <p class="status">Loading…</p>
  {:else if card.state.status === "error"}
    <p class="status status-error">Something went wrong: {card.state.message}</p>
  {:else}
    {@const detail = card.state.data}
    <article class="card-detail">
      <div class="card-detail-image">
        {#if detail.images.large || detail.images.small}
          <img src={detail.images.large ?? detail.images.small} alt={detail.name} />
        {:else}
          <div class="card-image-placeholder">No image</div>
        {/if}
      </div>
      <div class="card-detail-body">
        <h1>{detail.name}</h1>
        <dl class="card-facts">
          {#if detail.number !== null}
            <dt>Number</dt>
            <dd>#{detail.number}</dd>
          {/if}
          {#if detail.rarity !== null}
            <dt>Rarity</dt>
            <dd>{detail.rarity}</dd>
          {/if}
          <dt>Set</dt>
          <dd>
            <a href={`/g/${encodeURIComponent(data.game)}/s/${encodeURIComponent(detail.set.key)}`}>
              {detail.set.name}
            </a>
          </dd>
          <dt>Language</dt>
          <dd>{detail.language}</dd>
          {#if detail.tcgplayerId !== null}
            <dt>TCGplayer ID</dt>
            <dd>{detail.tcgplayerId}</dd>
          {/if}
        </dl>
        <h2>Variants</h2>
        {#if detail.variants.length === 0}
          <p class="status">No variants.</p>
        {:else}
          <table class="variant-table">
            <thead>
              <tr>
                <th>Variant</th>
                <th>Price</th>
              </tr>
            </thead>
            <tbody>
              {#each detail.variants as variant (variant.id)}
                <tr>
                  <td>{variant.name}</td>
                  <td>{variant.price !== null ? formatPrice(variant.price) : "—"}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    </article>
  {/if}
</section>
