import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";
import Pricing from "$lib/components/Pricing.svelte";
import { pricingTiers } from "$lib/pricing";
import { appUrl } from "$lib/config";
import LandingPage from "./+page.svelte";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("pricing section", () => {
  it("renders every tier with its name, price, and features", () => {
    render(Pricing);

    for (const tier of pricingTiers) {
      expect(screen.getByText(tier.name)).toBeTruthy();
      expect(screen.getByText(tier.price)).toBeTruthy();
      for (const feature of tier.features) {
        expect(screen.getByText(feature)).toBeTruthy();
      }
    }
  });

  it("points every tier CTA at PUBLIC_APP_URL", () => {
    vi.stubEnv("PUBLIC_APP_URL", "https://app.tcg-catalog.test");

    render(Pricing);

    for (const tier of pricingTiers) {
      const cta = screen.getByText(tier.cta).closest("a");
      expect(cta?.getAttribute("href")).toBe("https://app.tcg-catalog.test");
    }
  });
});

describe("landing page CTAs", () => {
  it("uses PUBLIC_APP_URL for the primary call to action", () => {
    vi.stubEnv("PUBLIC_APP_URL", "https://app.tcg-catalog.test");

    render(LandingPage);

    const getStarted = screen.getByText("Get started").closest("a");
    expect(getStarted?.getAttribute("href")).toBe("https://app.tcg-catalog.test");
  });

  it("falls back to the placeholder app URL when the env var is unset", () => {
    vi.stubEnv("PUBLIC_APP_URL", "");
    // An empty string is falsy config; appUrl only falls back on undefined,
    // so simulate "unset" by deleting the stubbed value.
    vi.unstubAllEnvs();
    delete (import.meta.env as Record<string, unknown>)["PUBLIC_APP_URL"];

    expect(appUrl()).toBe("https://app.example.com");
  });
});
