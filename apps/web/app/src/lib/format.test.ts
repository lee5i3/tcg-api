import { describe, expect, it } from "vitest";
import { cardMeta } from "./format";

describe("cardMeta", () => {
  it("shows the number and rarity when both are present", () => {
    expect(cardMeta({ number: "25", rarity: "Rare" })).toBe("#25 · Rare");
  });

  it("shows only the number when rarity is missing", () => {
    expect(cardMeta({ number: "104b", rarity: null })).toBe("#104b");
  });
});
