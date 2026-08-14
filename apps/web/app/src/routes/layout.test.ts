import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import { ApiError, type User } from "@tcg/api-client";
import { clearSession, TOKEN_KEY } from "$lib/auth.svelte";
import Layout from "./+layout.svelte";

const { me, goto, pageState } = vi.hoisted(() => ({
  me: vi.fn<() => Promise<User>>(),
  goto: vi.fn(),
  pageState: { url: new URL("http://localhost/") },
}));

vi.mock("$app/navigation", () => ({
  goto: (...args: unknown[]) => goto(...args),
}));

vi.mock("$app/state", () => ({
  page: pageState,
}));

vi.mock("@tcg/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tcg/api-client")>();
  return {
    ...actual,
    createClient: () => ({ ...({} as import("@tcg/api-client").TcgApiClient), me }),
  };
});

const user: User = {
  id: "u1",
  email: "ash@example.com",
  name: "Ash Ketchum",
  createdAt: "2026-01-01T00:00:00Z",
};

const children = createRawSnippet(() => ({
  render: () => "<p>catalog content</p>",
}));

function renderLayout(): ReturnType<typeof render> {
  return render(Layout, { props: { children } });
}

describe("root layout auth guard", () => {
  beforeEach(() => {
    me.mockReset();
    goto.mockReset();
    localStorage.clear();
    clearSession();
    pageState.url = new URL("http://localhost/");
  });

  it("redirects unauthenticated users to /login and preserves the return path", async () => {
    pageState.url = new URL("http://localhost/g/pokemon?query=char");

    renderLayout();

    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith(
        `/login?redirect=${encodeURIComponent("/g/pokemon?query=char")}`,
        { replaceState: true },
      ),
    );
    expect(screen.queryByText("catalog content")).toBeNull();
  });

  it("redirects the root path to a plain /login", async () => {
    renderLayout();

    await waitFor(() => expect(goto).toHaveBeenCalledWith("/login", { replaceState: true }));
    expect(screen.queryByText("catalog content")).toBeNull();
  });

  it("renders public routes without a session and without redirecting", async () => {
    pageState.url = new URL("http://localhost/login");

    renderLayout();

    expect(await screen.findByText("catalog content")).toBeTruthy();
    expect(goto).not.toHaveBeenCalled();
    expect(me).not.toHaveBeenCalled();
    expect(screen.queryByText("Sign out")).toBeNull();
  });

  it("shows the signed-in user and signs out via the header button", async () => {
    localStorage.setItem(TOKEN_KEY, "jwt-123");
    me.mockResolvedValue(user);

    renderLayout();

    expect(await screen.findByText("Ash Ketchum")).toBeTruthy();
    expect(screen.getByText("catalog content")).toBeTruthy();

    await fireEvent.click(screen.getByText("Sign out"));

    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    await waitFor(() => expect(goto).toHaveBeenCalledWith("/login"));
  });

  it("falls back to the email when the user has no name", async () => {
    localStorage.setItem(TOKEN_KEY, "jwt-123");
    me.mockResolvedValue({ ...user, name: null });

    renderLayout();

    expect(await screen.findByText("ash@example.com")).toBeTruthy();
  });

  it("clears an invalid stored token (401 from me) and redirects to /login", async () => {
    localStorage.setItem(TOKEN_KEY, "expired-jwt");
    me.mockRejectedValue(new ApiError(401, "invalid token"));
    pageState.url = new URL("http://localhost/g/pokemon");

    renderLayout();

    await waitFor(() => expect(localStorage.getItem(TOKEN_KEY)).toBeNull());
    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith(
        `/login?redirect=${encodeURIComponent("/g/pokemon")}`,
        { replaceState: true },
      ),
    );
  });

  it("keeps the session when me() fails with a non-401 error", async () => {
    localStorage.setItem(TOKEN_KEY, "jwt-123");
    me.mockRejectedValue(new ApiError(500, "server exploded"));

    renderLayout();

    expect(await screen.findByText("catalog content")).toBeTruthy();
    expect(localStorage.getItem(TOKEN_KEY)).toBe("jwt-123");
    expect(goto).not.toHaveBeenCalled();
  });
});
