import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

// vitest runs with the package root as cwd (see the "test" script).
const tokensCss = readFileSync(resolve(process.cwd(), "src/tokens.css"), "utf8");

const CORE_TOKENS = [
  // Accent and dark-panel colors
  "--ds-color-accent",
  "--ds-color-accent-strong",
  "--ds-color-on-accent",
  "--ds-gradient-accent",
  "--ds-color-dark",
  "--ds-color-on-dark",
  // Neutrals
  "--ds-color-bg",
  "--ds-color-surface",
  "--ds-color-border",
  "--ds-color-ink",
  "--ds-color-muted",
  // Semantic
  "--ds-color-success",
  "--ds-color-danger",
  "--ds-color-up",
  "--ds-color-down",
  // Typography
  "--ds-font-display",
  "--ds-font-body",
  "--ds-font-mono",
  "--ds-text-base",
  "--ds-weight-bold",
  "--ds-leading-normal",
  "--ds-tracking-tight",
  // Spacing, radii, shadows, focus
  "--ds-space-1",
  "--ds-space-4",
  "--ds-space-8",
  "--ds-radius-sm",
  "--ds-radius-md",
  "--ds-radius-lg",
  "--ds-shadow-md",
  "--ds-focus-ring",
];

describe("tokens.css", () => {
  it("parses as CSS with a :root rule", () => {
    const style = document.createElement("style");
    style.textContent = tokensCss;
    document.head.appendChild(style);

    const sheet = style.sheet;
    expect(sheet).toBeTruthy();
    expect(sheet!.cssRules.length).toBeGreaterThan(0);
    const rootRule = [...sheet!.cssRules].find(
      (rule) => rule instanceof CSSStyleRule && rule.selectorText === ":root",
    );
    expect(rootRule).toBeTruthy();

    style.remove();
  });

  it("has balanced braces", () => {
    const opens = tokensCss.match(/\{/g)?.length ?? 0;
    const closes = tokensCss.match(/\}/g)?.length ?? 0;
    expect(opens).toBeGreaterThan(0);
    expect(opens).toBe(closes);
  });

  it.each(CORE_TOKENS)("defines %s", (token) => {
    expect(tokensCss).toMatch(new RegExp(`${token}\\s*:`));
  });
});
