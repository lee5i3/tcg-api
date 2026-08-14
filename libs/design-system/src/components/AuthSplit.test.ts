import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import AuthSplit from "./AuthSplit.svelte";

const children = createRawSnippet(() => ({
  render: () => "<form data-testid='form'>form content</form>",
}));

const brand = createRawSnippet(() => ({
  render: () => "<span>Brand lockup</span>",
}));

const aside = createRawSnippet(() => ({
  render: () => "<p>Marketing moment</p>",
}));

describe("AuthSplit", () => {
  it("renders the form content inside the vertically-centered column", () => {
    render(AuthSplit, { props: { children } });

    const form = screen.getByTestId("form");
    expect(form.closest(".ds-authsplit-main")).toBeTruthy();
    expect(form.closest(".ds-authsplit-form")).toBeTruthy();
  });

  it("stays a single column without an aside", () => {
    const { container } = render(AuthSplit, { props: { children } });

    const root = container.querySelector(".ds-authsplit") as HTMLElement;
    expect(root.classList.contains("ds-authsplit-two")).toBe(false);
    expect(container.querySelector("aside")).toBeNull();
  });

  it("renders the brand lockup in a header at the top of the form column", () => {
    render(AuthSplit, { props: { children, brand } });

    const lockup = screen.getByText("Brand lockup");
    expect(lockup.closest("header.ds-authsplit-brand")).toBeTruthy();
  });

  it("renders the aside as a second column when provided", () => {
    const { container } = render(AuthSplit, { props: { children, aside } });

    const root = container.querySelector(".ds-authsplit") as HTMLElement;
    expect(root.classList.contains("ds-authsplit-two")).toBe(true);
    const marketing = screen.getByText("Marketing moment");
    expect(marketing.closest("aside.ds-authsplit-aside")).toBeTruthy();
  });

  it("omits the brand header when no brand snippet is given", () => {
    const { container } = render(AuthSplit, { props: { children } });

    expect(container.querySelector(".ds-authsplit-brand")).toBeNull();
  });
});
