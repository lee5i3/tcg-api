import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import TextField from "./TextField.svelte";

describe("TextField", () => {
  it("associates the label with the input", () => {
    render(TextField, { props: { label: "Email", type: "email", name: "email" } });

    const input = screen.getByLabelText("Email") as HTMLInputElement;
    expect(input.tagName).toBe("INPUT");
    expect(input.type).toBe("email");
    expect(input.name).toBe("email");
  });

  it("renders the hint as part of the accessible label", () => {
    render(TextField, { props: { label: "Name", hint: "(optional)" } });

    expect(screen.getByLabelText(/^Name/)).toBeTruthy();
    expect(screen.getByText("(optional)")).toBeTruthy();
  });

  it("accepts typed input", async () => {
    render(TextField, { props: { label: "Key" } });

    const input = screen.getByLabelText("Key") as HTMLInputElement;
    await fireEvent.input(input, { target: { value: "pokemon" } });
    expect(input.value).toBe("pokemon");
  });

  it("renders an error with role=alert and marks the input invalid", () => {
    render(TextField, { props: { label: "Email", error: "Enter a valid email." } });

    const alert = screen.getByRole("alert");
    expect(alert).toHaveProperty("textContent", "Enter a valid email.");
    const input = screen.getByLabelText("Email");
    expect(input.getAttribute("aria-invalid")).toBe("true");
    expect(input.getAttribute("aria-describedby")).toBe(alert.id);
  });

  it("renders no alert and no aria-invalid without an error", () => {
    render(TextField, { props: { label: "Email" } });

    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.getByLabelText("Email").getAttribute("aria-invalid")).toBeNull();
  });

  it("forwards attributes like disabled and required", () => {
    render(TextField, { props: { label: "Key", required: true, disabled: true } });

    const input = screen.getByLabelText("Key") as HTMLInputElement;
    expect(input.required).toBe(true);
    expect(input.disabled).toBe(true);
  });

  it("renders an optional decorative leading icon and keeps the label wired up", () => {
    const icon = createRawSnippet(() => ({
      render: () => "<svg data-testid='mail-icon'></svg>",
    }));

    const { container } = render(TextField, { props: { label: "Email", icon } });

    const holder = container.querySelector(".ds-field-icon") as HTMLElement;
    expect(holder).toBeTruthy();
    expect(holder.getAttribute("aria-hidden")).toBe("true");
    expect(screen.getByTestId("mail-icon")).toBeTruthy();

    const input = screen.getByLabelText("Email");
    expect(input.classList.contains("ds-field-input-iconed")).toBe(true);
  });

  it("renders no icon holder and no icon padding by default", () => {
    const { container } = render(TextField, { props: { label: "Email" } });

    expect(container.querySelector(".ds-field-icon")).toBeNull();
    const input = screen.getByLabelText("Email");
    expect(input.classList.contains("ds-field-input-iconed")).toBe(false);
  });

  it("applies dark-panel styling via tone=\"dark\"", () => {
    const { container } = render(TextField, { props: { label: "Email", tone: "dark" } });

    const field = container.querySelector(".ds-field") as HTMLElement;
    expect(field.classList.contains("ds-field-dark")).toBe(true);
  });

  it("stays on the light tone by default", () => {
    const { container } = render(TextField, { props: { label: "Email" } });

    const field = container.querySelector(".ds-field") as HTMLElement;
    expect(field.classList.contains("ds-field-dark")).toBe(false);
  });
});
