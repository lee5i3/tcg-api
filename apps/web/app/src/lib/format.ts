/** Meta line shown under a card tile, e.g. "#123 · Rare". */
export function cardMeta(card: { number: string; rarity: string | null }): string {
  const number = `#${card.number}`;
  return card.rarity ? `${number} · ${card.rarity}` : number;
}
