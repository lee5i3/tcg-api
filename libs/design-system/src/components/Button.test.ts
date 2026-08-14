import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import Button from "./Button.svelte";

function label(text: string) {
  return createRawSnippet(() => ({ render: () => `<span>${text}</span>` }));
}

describe("Button", () => {
  it("renders a native button with the primary variant and md size by default", () => {
    render(Button, { props: { children: label("Save") } });

    const button = screen.getByRole("button", { name: "Save" });
    expect(button.tagName).toBe("BUTTON");
    expect(button.className).toContain("ds-btn");
    expect(button.className).toContain("ds-btn-primary");
    expect(button.className).not.toContain("ds-btn-sm");
  });

  it.each(["primary", "secondary", "danger", "ghost", "inverse", "paper"] as const)(
    "applies the %s variant class",
    (variant) => {
      render(Button, { props: { variant, children: label("Go") } });

      expect(screen.getByRole("button").className).toContain(`ds-btn-${variant}`);
    },
  );

  it("applies the small size class", () => {
    render(Button, { props: { size: "sm", children: label("Edit") } });

    expect(screen.getByRole("button").className).toContain("ds-btn-sm");
  });

  it("renders an anchor when href is given, keeping the button classes", () => {
    render(Button, {
      props: { href: "/pricing", variant: "secondary", children: label("See pricing") },
    });

    const link = screen.getByRole("link", { name: "See pricing" });
    expect(link.getAttribute("href")).toBe("/pricing");
    expect(link.className).toContain("ds-btn-secondary");
  });

  it("forwards button attributes and click handlers", async () => {
    const onclick = vi.fn();
    render(Button, {
      props: { type: "submit", onclick, children: label("Submit") },
    });

    const button = screen.getByRole("button", { name: "Submit" });
    expect(button.getAttribute("type")).toBe("submit");
    await fireEvent.click(button);
    expect(onclick).toHaveBeenCalledTimes(1);
  });

  it("supports the disabled attribute", () => {
    render(Button, { props: { disabled: true, children: label("Wait") } });

    expect((screen.getByRole("button") as HTMLButtonElement).disabled).toBe(true);
  });
});
