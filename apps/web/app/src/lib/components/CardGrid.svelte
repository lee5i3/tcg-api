<script lang="ts">
  import { formatPrice, lowestPrice, type CardSummary } from "$lib/api";
  import { cardMeta } from "$lib/format";

  let { game, cards }: { game: string; cards: CardSummary[] } = $props();
</script>

{#if cards.length === 0}
  <p class="status">No cards found.</p>
{:else}
  <ul class="card-grid">
    {#each cards as card (card.id)}
      {@const price = lowestPrice(card.variants)}
      <li>
        <a class="card-tile" href={`/g/${encodeURIComponent(game)}/c/${card.id}`}>
          {#if card.image}
            <img class="card-image" src={card.image} alt={card.name} loading="lazy" />
          {:else}
            <div class="card-image card-image-placeholder">No image</div>
          {/if}
          <div class="card-tile-body">
            <span class="card-name">{card.name}</span>
            <span class="card-meta">{cardMeta(card)}</span>
            {#if price !== null}
              <span class="card-price">{formatPrice(price)}</span>
            {/if}
          </div>
        </a>
      </li>
    {/each}
  </ul>
{/if}
