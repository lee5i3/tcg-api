import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { ApiError, type AuthSession } from "@tcg/api-client";
import { clearSession, TOKEN_KEY } from "$lib/auth.svelte";
import LoginPage from "./+page.svelte";

const { login, providers, goto, pageState } = vi.hoisted(() => ({
  login: vi.fn<(input: { email: string; password: string }) => Promise<AuthSession>>(),
  providers: vi.fn<() => Promise<{ id: string; label: string }[]>>(),
  goto: vi.fn(),
  pageState: { url: new URL("http://localhost/login") },
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
    createClient: () => ({ ...({} as import("@tcg/api-client").TcgApiClient), login, providers }),
  };
});

const session: AuthSession = {
  token: "jwt-123",
  user: {
    id: "u1",
    email: "ash@example.com",
    name: "Ash Ketchum",
    createdAt: "2026-01-01T00:00:00Z",
  },
};

async function fillAndSubmit(email: string, password: string): Promise<void> {
  const emailInput = screen.getByLabelText("Email");
  await fireEvent.input(emailInput, { target: { value: email } });
  await fireEvent.input(screen.getByLabelText("Password"), { target: { value: password } });
  await fireEvent.submit(emailInput.closest("form") as HTMLFormElement);
}

describe("login page", () => {
  beforeEach(() => {
    login.mockReset();
    providers.mockReset();
    providers.mockResolvedValue([]);
    goto.mockReset();
    localStorage.clear();
    clearSession();
    pageState.url = new URL("http://localhost/login");
  });

  it("stores the token and redirects to the requested path on success", async () => {
    pageState.url = new URL("http://localhost/login?redirect=%2Fg%2Fpokemon");
    login.mockResolvedValue(session);

    render(LoginPage);
    await fillAndSubmit("ash@example.com", "pikachu123");

    await waitFor(() => expect(goto).toHaveBeenCalledWith("/g/pokemon"));
    expect(login).toHaveBeenCalledWith({ email: "ash@example.com", password: "pikachu123" });
    expect(localStorage.getItem(TOKEN_KEY)).toBe("jwt-123");
  });

  it("redirects to the catalog root when no return path was requested", async () => {
    login.mockResolvedValue(session);

    render(LoginPage);
    await fillAndSubmit("ash@example.com", "pikachu123");

    await waitFor(() => expect(goto).toHaveBeenCalledWith("/"));
  });

  it("shows 'invalid email or password' on a 401 and keeps no token", async () => {
    login.mockRejectedValue(new ApiError(401, "unauthorized"));

    render(LoginPage);
    await fillAndSubmit("ash@example.com", "wrong-pass");

    expect(await screen.findByText("invalid email or password")).toBeTruthy();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(goto).not.toHaveBeenCalled();
  });

  it("shows other ApiError messages inline", async () => {
    login.mockRejectedValue(new ApiError(400, "invalid email"));

    render(LoginPage);
    await fillAndSubmit("not-an-email", "pikachu123");

    expect(await screen.findByText("invalid email")).toBeTruthy();
  });

  it("offers a link to create an account", () => {
    render(LoginPage);

    const link = screen.getByText("Create an account");
    expect(link.getAttribute("href")).toBe("/register");
  });

  it("carries the return path over to the register link", () => {
    pageState.url = new URL("http://localhost/login?redirect=%2Fg%2Fpokemon");

    render(LoginPage);

    const link = screen.getByText("Create an account");
    expect(link.getAttribute("href")).toBe("/register?redirect=%2Fg%2Fpokemon");
  });

  it("shows a ?error= message (e.g. from a failed OAuth round-trip) inline", async () => {
    pageState.url = new URL("http://localhost/login?error=social%20sign-in%20failed");

    render(LoginPage);

    expect(await screen.findByText("social sign-in failed")).toBeTruthy();
  });

  it("renders one social button per provider, in fixed order, as /v1 start links", async () => {
    providers.mockResolvedValue([
      { id: "acme", label: "Acme ID" },
      { id: "apple", label: "Apple" },
      { id: "google", label: "Google" },
    ]);

    render(LoginPage);

    const google = await screen.findByText("Continue with Google");
    const googleLink = google.closest("a") as HTMLAnchorElement;
    expect(googleLink.getAttribute("href")).toBe("/v1/auth/oauth/google/start?redirect=%2F");
    // Full browser navigation: /v1/... is the API, not an app route.
    expect(googleLink.hasAttribute("data-sveltekit-reload")).toBe(true);

    // getAllByText returns elements in document order — known providers
    // first (google, then apple), unknown ones after.
    const labels = screen.getAllByText(/^Continue with/).map((el) => el.textContent?.trim());
    expect(labels).toEqual(["Continue with Google", "Continue with Apple", "Continue with Acme ID"]);
    expect(
      screen.getByText("Continue with Apple").closest("a")?.getAttribute("href"),
    ).toBe("/v1/auth/oauth/apple/start?redirect=%2F");
  });

  it("passes the sanitized ?redirect through to the start links", async () => {
    pageState.url = new URL("http://localhost/login?redirect=%2Fg%2Fpokemon");
    providers.mockResolvedValue([{ id: "google", label: "Google" }]);

    render(LoginPage);

    const link = (await screen.findByText("Continue with Google")).closest("a");
    expect(link?.getAttribute("href")).toBe("/v1/auth/oauth/google/start?redirect=%2Fg%2Fpokemon");
  });

  it("renders no social section when the provider list is empty", async () => {
    providers.mockResolvedValue([]);

    render(LoginPage);

    await waitFor(() => expect(providers).toHaveBeenCalled());
    expect(screen.queryByText("or continue with")).toBeNull();
    expect(screen.queryByText(/^Continue with/)).toBeNull();
  });

  it("renders no social section (and no error) when the providers fetch fails", async () => {
    providers.mockRejectedValue(new Error("network down"));

    render(LoginPage);

    await waitFor(() => expect(providers).toHaveBeenCalled());
    expect(screen.queryByText(/^Continue with/)).toBeNull();
  });
});
