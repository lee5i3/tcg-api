import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import { ApiError, type User } from "@tcg/api-client";
import { auth, clearSession, TOKEN_KEY } from "$lib/auth.svelte";
import CallbackPage from "./+page.svelte";

const { me, goto } = vi.hoisted(() => ({
  me: vi.fn<() => Promise<User>>(),
  goto: vi.fn(),
}));

vi.mock("$app/navigation", () => ({
  goto: (...args: unknown[]) => goto(...args),
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

const FAILURE = "/login?error=social%20sign-in%20failed";

describe("/auth/callback", () => {
  beforeEach(() => {
    me.mockReset();
    goto.mockReset();
    localStorage.clear();
    clearSession();
    window.location.hash = "";
  });

  it("shows a minimal signing-in state while working", () => {
    window.location.hash = "#token=jwt-oauth";
    me.mockReturnValue(new Promise(() => {})); // never resolves

    render(CallbackPage);

    expect(screen.getByText("Signing you in…")).toBeTruthy();
  });

  it("stores the fragment token, validates it, and redirects to the target", async () => {
    window.location.hash = "#token=jwt-oauth&redirect=%2Fg%2Fpokemon";
    me.mockResolvedValue(user);

    render(CallbackPage);

    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith("/g/pokemon", { replaceState: true }),
    );
    expect(localStorage.getItem(TOKEN_KEY)).toBe("jwt-oauth");
    expect(auth.user).toEqual(user);
    expect(me).toHaveBeenCalled();
  });

  it("falls back to the catalog root when the redirect is missing or unsafe", async () => {
    window.location.hash = "#token=jwt-oauth&redirect=https%3A%2F%2Fevil.example";
    me.mockResolvedValue(user);

    render(CallbackPage);

    await waitFor(() => expect(goto).toHaveBeenCalledWith("/", { replaceState: true }));
  });

  it("clears the session and returns to /login with an error on a bad token", async () => {
    window.location.hash = "#token=bad-token&redirect=%2F";
    me.mockRejectedValue(new ApiError(401, "invalid token"));

    render(CallbackPage);

    await waitFor(() => expect(goto).toHaveBeenCalledWith(FAILURE, { replaceState: true }));
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(auth.isAuthenticated).toBe(false);
  });

  it("goes straight to /login with an error when the fragment has no token", async () => {
    window.location.hash = "#redirect=%2Fg%2Fpokemon";

    render(CallbackPage);

    await waitFor(() => expect(goto).toHaveBeenCalledWith(FAILURE, { replaceState: true }));
    expect(me).not.toHaveBeenCalled();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
  });
});
