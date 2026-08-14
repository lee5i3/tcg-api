import { ApiError, type AuthSession, type User } from "@tcg/api-client";
import { api, configureApi } from "$lib/api";

/** localStorage key the user's JWT is persisted under (survives reloads). */
export const TOKEN_KEY = "tcg-app-token";

interface AuthState {
  token: string | null;
  user: User | null;
}

const state = $state<AuthState>({ token: null, user: null });

/** Reactive view of the current session. */
export const auth = {
  get token(): string | null {
    return state.token;
  },
  get user(): User | null {
    return state.user;
  },
  get isAuthenticated(): boolean {
    return state.token !== null;
  },
};

/**
 * Restores a persisted token on app start (the user object is filled in
 * later by `validateSession`). Safe to call more than once.
 */
export function initAuth(): void {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token && token !== state.token) {
    state.token = token;
    state.user = null;
    configureApi(token);
  }
}

/** Persists a fresh login/register session and re-keys the API client. */
export function setSession(session: AuthSession): void {
  localStorage.setItem(TOKEN_KEY, session.token);
  state.token = session.token;
  state.user = session.user;
  configureApi(session.token);
}

/**
 * Persists a session from a bare token (OAuth callbacks hand us only the
 * JWT), then validates it via GET /v1/auth/me to fill in the user. On any
 * failure the half-open session is cleared and the error rethrown.
 */
export async function setSessionFromToken(token: string): Promise<void> {
  localStorage.setItem(TOKEN_KEY, token);
  state.token = token;
  state.user = null;
  configureApi(token);
  try {
    state.user = await api.me();
  } catch (error) {
    clearSession();
    throw error;
  }
}

/** Signs out: forgets the token and returns the API client to anonymous. */
export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  state.token = null;
  state.user = null;
  configureApi();
}

/**
 * Sanitizes a `?redirect=` query value: only same-app absolute paths are
 * allowed; anything else falls back to the catalog root.
 */
export function sanitizeRedirect(raw: string | null): string {
  if (raw && raw.startsWith("/") && !raw.startsWith("//")) {
    return raw;
  }
  return "/";
}

/**
 * Validates the stored token against GET /v1/auth/me. On 401 the session is
 * cleared (the layout guard then redirects to /login). Other errors (network,
 * 5xx) keep the session so a flaky API doesn't log users out.
 */
export async function validateSession(): Promise<boolean> {
  try {
    state.user = await api.me();
    return true;
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      clearSession();
      return false;
    }
    return true;
  }
}
