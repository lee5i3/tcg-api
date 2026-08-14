import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { ApiError, type AuthSession, type RegisterInput } from "@tcg/api-client";
import { clearSession, TOKEN_KEY } from "$lib/auth.svelte";
import RegisterPage from "./+page.svelte";

const { register, providers, goto, pageState } = vi.hoisted(() => ({
  register: vi.fn<(input: { email: string; password: string; name?: string }) => Promise<AuthSession>>(),
  providers: vi.fn<() => Promise<{ id: string; label: string }[]>>(),
  goto: vi.fn(),
  pageState: { url: new URL("http://localhost/register") },
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
    createClient: () => ({ ...({} as import("@tcg/api-client").TcgApiClient), register, providers }),
  };
});

const session: AuthSession = {
  token: "jwt-456",
  user: {
    id: "u2",
    email: "misty@example.com",
    name: "Misty",
    createdAt: "2026-01-01T00:00:00Z",
  },
};

interface FormValues {
  name?: string;
  email: string;
  password: string;
  confirmPassword: string;
}

async function fillAndSubmit(values: FormValues): Promise<void> {
  const emailInput = screen.getByLabelText("Email");
  if (values.name !== undefined) {
    await fireEvent.input(screen.getByLabelText(/^Name/), { target: { value: values.name } });
  }
  await fireEvent.input(emailInput, { target: { value: values.email } });
  await fireEvent.input(screen.getByLabelText("Password"), {
    target: { value: values.password },
  });
  await fireEvent.input(screen.getByLabelText("Confirm password"), {
    target: { value: values.confirmPassword },
  });
  await fireEvent.submit(emailInput.closest("form") as HTMLFormElement);
}

describe("register page", () => {
  beforeEach(() => {
    register.mockReset();
    providers.mockReset();
    providers.mockResolvedValue([]);
    goto.mockReset();
    localStorage.clear();
    clearSession();
    pageState.url = new URL("http://localhost/register");
  });

  it("blocks submission client-side when the passwords do not match", async () => {
    render(RegisterPage);
    await fillAndSubmit({
      email: "misty@example.com",
      password: "starmie123",
      confirmPassword: "starmie124",
    });

    expect(await screen.findByText("Passwords do not match.")).toBeTruthy();
    expect(register).not.toHaveBeenCalled();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
  });

  it("registers, stores the token, and redirects on success", async () => {
    register.mockResolvedValue(session);

    render(RegisterPage);
    await fillAndSubmit({
      name: "Misty",
      email: "misty@example.com",
      password: "starmie123",
      confirmPassword: "starmie123",
    });

    await waitFor(() => expect(goto).toHaveBeenCalledWith("/"));
    const expected: RegisterInput = {
      email: "misty@example.com",
      password: "starmie123",
      name: "Misty",
    };
    expect(register).toHaveBeenCalledWith(expected);
    expect(localStorage.getItem(TOKEN_KEY)).toBe("jwt-456");
  });

  it("omits the name field when left blank and honours the return path", async () => {
    pageState.url = new URL("http://localhost/register?redirect=%2Fg%2Fpokemon");
    register.mockResolvedValue(session);

    render(RegisterPage);
    await fillAndSubmit({
      email: "misty@example.com",
      password: "starmie123",
      confirmPassword: "starmie123",
    });

    await waitFor(() => expect(goto).toHaveBeenCalledWith("/g/pokemon"));
    expect(register).toHaveBeenCalledWith({
      email: "misty@example.com",
      password: "starmie123",
    });
  });

  it("shows API errors such as a taken email inline", async () => {
    register.mockRejectedValue(new ApiError(409, "email already registered"));

    render(RegisterPage);
    await fillAndSubmit({
      email: "misty@example.com",
      password: "starmie123",
      confirmPassword: "starmie123",
    });

    expect(await screen.findByText("email already registered")).toBeTruthy();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(goto).not.toHaveBeenCalled();
  });

  it("links back to the sign-in page", () => {
    render(RegisterPage);

    const link = screen.getByText("Sign in instead");
    expect(link.getAttribute("href")).toBe("/login");
  });

  it("carries the return path over to the sign-in link", () => {
    pageState.url = new URL("http://localhost/register?redirect=%2Fg%2Fpokemon");

    render(RegisterPage);

    const link = screen.getByText("Sign in instead");
    expect(link.getAttribute("href")).toBe("/login?redirect=%2Fg%2Fpokemon");
  });

  it("renders social sign-in buttons from the configured providers", async () => {
    pageState.url = new URL("http://localhost/register?redirect=%2Fg%2Fpokemon");
    providers.mockResolvedValue([{ id: "google", label: "Google" }]);

    render(RegisterPage);

    const link = (await screen.findByText("Continue with Google")).closest("a");
    expect(link?.getAttribute("href")).toBe("/v1/auth/oauth/google/start?redirect=%2Fg%2Fpokemon");
  });

  it("renders no social section when no provider is configured", async () => {
    render(RegisterPage);

    await waitFor(() => expect(providers).toHaveBeenCalled());
    expect(screen.queryByText(/^Continue with/)).toBeNull();
  });
});
