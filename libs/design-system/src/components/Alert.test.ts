import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import Alert from "./Alert.svelte";

function message(text: string) {
  return createRawSnippet(() => ({ render: () => `<span>${text}</span>` }));
}

describe("Alert", () => {
  it("renders an error alert by default with role=alert", () => {
    render(Alert, { props: { children: message("Invalid token.") } });

    const alert = screen.getByRole("alert");
    expect(alert.textContent).toBe("Invalid token.");
    expect(alert.className).toContain("ds-alert-error");
  });

  it("renders the success variant", () => {
    render(Alert, { props: { variant: "success", children: message("Created set.") } });

    const alert = screen.getByRole("alert");
    expect(alert.textContent).toBe("Created set.");
    expect(alert.className).toContain("ds-alert-success");
    expect(alert.className).not.toContain("ds-alert-error");
  });
});
