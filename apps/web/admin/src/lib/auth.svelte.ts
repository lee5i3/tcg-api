import { createClient, type TcgApiClient } from "@tcg/api-client";

// Empty base URL means same-origin: the admin site is served behind a
// reverse proxy together with the API, so requests default to relative
// /v1/... URLs. Set VITE_API_URL only to point at a remote API in dev.
const BASE_URL = import.meta.env.VITE_API_URL ?? "";

const TOKEN_STORAGE_KEY = "tcg-admin-token";

function readStoredToken(): string | null {
  if (typeof sessionStorage === "undefined") {
    return null;
  }
  return sessionStorage.getItem(TOKEN_STORAGE_KEY);
}

let token = $state<string | null>(readStoredToken());

export const auth = {
  /** Current admin token, or null when logged out. */
  get token(): string | null {
    return token;
  },

  /** API client bound to the current token. */
  get client(): TcgApiClient {
    return createClient(BASE_URL, token ? { token } : undefined);
  },

  /**
   * Validates the token against POST /v1/auth/check. On success stores it
   * (sessionStorage) and enters the app; returns an error message otherwise.
   */
  async login(candidate: string): Promise<string | null> {
    const trimmed = candidate.trim();
    if (!trimmed) {
      return "Enter a token.";
    }
    try {
      const valid = await createClient(BASE_URL, { token: trimmed }).authCheck();
      if (!valid) {
        return "Invalid token.";
      }
    } catch (error) {
      return error instanceof Error ? error.message : "Token check failed.";
    }
    sessionStorage.setItem(TOKEN_STORAGE_KEY, trimmed);
    token = trimmed;
    return null;
  },

  /** Clears the stored token and returns to the login screen. */
  logout(): void {
    sessionStorage.removeItem(TOKEN_STORAGE_KEY);
    token = null;
  },
};
