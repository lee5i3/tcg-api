import { describe, expect, it } from "vitest";
import { pricingTiers } from "./pricing";

describe("pricing tiers", () => {
  it("defines the three placeholder tiers in order", () => {
    expect(pricingTiers.map((tier) => tier.id)).toEqual(["free", "collector", "pro"]);
    expect(pricingTiers.map((tier) => tier.name)).toEqual(["Free", "Collector", "Pro"]);
    expect(pricingTiers.map((tier) => tier.price)).toEqual(["$0", "$9", "$29"]);
  });

  it("highlights exactly one tier", () => {
    expect(pricingTiers.filter((tier) => tier.highlighted).map((tier) => tier.id)).toEqual([
      "collector",
    ]);
  });

  it("gives every tier a CTA and at least three features", () => {
    for (const tier of pricingTiers) {
      expect(tier.cta.length).toBeGreaterThan(0);
      expect(tier.features.length).toBeGreaterThanOrEqual(3);
    }
  });
});
