export interface FaqEntry {
  question: string;
  answer: string;
}

export const faqEntries: FaqEntry[] = [
  {
    question: "Which games are covered?",
    answer:
      "The catalog is multi-game by design. Pokémon is the most complete today, and new games are added as their catalogs are imported — each with full set lists and per-printing variants.",
  },
  {
    question: "Where do the prices come from?",
    answer:
      "Prices are market data refreshed on a schedule by our pricing jobs. For every card we show the lowest current price of each variant — holo, reverse, first edition, and so on.",
  },
  {
    question: "Do I need an account to browse?",
    answer:
      "No. The catalog, set lists, search, and live prices are free and open — just open the app. Accounts arrive together with collection tracking.",
  },
  {
    question: "Can I buy or sell cards here?",
    answer:
      "No — TCG Catalog is a reference, not a marketplace. Prices are market estimates to help you value a collection, not offers to buy.",
  },
];
