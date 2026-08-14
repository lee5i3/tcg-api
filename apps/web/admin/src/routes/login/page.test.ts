import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/svelte";
import LoginPage from "./+page.svelte";
import { auth } from "$lib/auth.svelte";

const { authCheck, goto } = vi.hoisted(() => ({
  authCheck: vi.fn<(token: string | undefined) => Promise<boolean>>(),
  goto: vi.fn(),
}));

vi.mock("$app/navigation", () => ({
  goto: (...args: unknown[]) => goto(...args),
}));

vi.mock("@tcg/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tcg/api-client")>();
  return {
    ...actual,
    createClient: (_baseUrl: string, options?: { token?: string }) =>
      ({
        authCheck: () => authCheck(options?.token),
      }) as unknown as import("@tcg/api-client").TcgApiClient,
  };
});

async function submitToken(token: string) {
  const input = screen.getByLabelText("Admin token");
  await fireEvent.input(input, { target: { value: token } });
  const form = input.closest("form");
  expect(form).toBeTruthy();
  await fireEvent.submit(form as HTMLFormElement);
}

describe("login page", () => {
  beforeEach(() => {
    sessionStorage.clear();
    auth.logout();
    authCheck.mockReset();
    goto.mockReset();
  });

  it("rejects a bad token and shows an inline error", async () => {
    authCheck.mockResolvedValue(false);

    render(LoginPage);
    await submitToken("wrong-token");

    expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Invalid token.");
    expect(authCheck).toHaveBeenCalledWith("wrong-token");
    expect(auth.token).toBeNull();
    expect(sessionStorage.getItem("tcg-admin-token")).toBeNull();
    expect(goto).not.toHaveBeenCalled();
  });

  it("requires a non-empty token", async () => {
    render(LoginPage);
    await submitToken("   ");

    expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Enter a token.");
    expect(authCheck).not.toHaveBeenCalled();
  });

  it("stores a good token in sessionStorage and navigates to the dashboard", async () => {
    authCheck.mockResolvedValue(true);

    render(LoginPage);
    await submitToken("valid-token");

    await vi.waitFor(() => {
      expect(goto).toHaveBeenCalledWith("/", { replaceState: true });
    });
    expect(authCheck).toHaveBeenCalledWith("valid-token");
    expect(auth.token).toBe("valid-token");
    expect(sessionStorage.getItem("tcg-admin-token")).toBe("valid-token");
  });

  it("surfaces API errors from the auth check", async () => {
    authCheck.mockRejectedValue(new Error("Request to /v1/auth/check failed with status 500"));

    render(LoginPage);
    await submitToken("any-token");

    expect(await screen.findByRole("alert")).toHaveProperty(
      "textContent",
      "Request to /v1/auth/check failed with status 500",
    );
    expect(auth.token).toBeNull();
  });
});
