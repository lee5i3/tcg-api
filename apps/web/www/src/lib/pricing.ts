/**
 * Pricing tiers shown on the landing page.
 *
 * Placeholder copy: paid plans are not purchasable yet (see TODO(signup)
 * in config.ts) — the structure is real, the details are subject to change.
 */
export interface PricingTier {
  id: string;
  name: string;
  price: string;
  period: string;
  tagline: string;
  features: string[];
  cta: string;
  highlighted: boolean;
}

export const pricingTiers: PricingTier[] = [
  {
    id: "free",
    name: "Free",
    price: "$0",
    period: "forever",
    tagline: "Browse the whole catalog.",
    features: [
      "Every game, set, and printing",
      "Live lowest price per variant",
      "Card search across a game",
      "No account needed",
    ],
    cta: "Open the app",
    highlighted: false,
  },
  {
    id: "collector",
    name: "Collector",
    price: "$9",
    period: "per month",
    tagline: "Put a number on your binder.",
    features: [
      "Everything in Free",
      "Track your collection card by card",
      "Collection value dashboard",
      "Price alerts on cards you watch",
    ],
    cta: "Start collecting",
    highlighted: true,
  },
  {
    id: "pro",
    name: "Pro",
    price: "$29",
    period: "per month",
    tagline: "For stores and serious traders.",
    features: [
      "Everything in Collector",
      "Price history charts",
      "Bulk import and export",
      "API access",
    ],
    cta: "Go Pro",
    highlighted: false,
  },
];
