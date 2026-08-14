import { createClient, type TcgApiClient } from "@tcg/api-client";

export type {
  Game,
  SetSummary,
  CardSummary,
  CardDetail,
  CardVariant,
  CardImages,
  TcgApiClient,
  User,
  AuthSession,
  AuthProvider,
} from "@tcg/api-client";
export { ApiError, lowestPrice, formatPrice } from "@tcg/api-client";

// Empty base URL means same-origin: the site is served behind CloudFront
// together with the API, so requests default to relative /v1/... URLs.
// Set VITE_API_URL only to point at a remote API in dev (see .env.example).
const baseUrl: string = import.meta.env.VITE_API_URL ?? "";

let client: TcgApiClient = createClient(baseUrl);

/**
 * Recreates the underlying client, attaching the signed-in user's JWT as a
 * bearer token (or dropping it when `token` is undefined).
 */
export function configureApi(token?: string): void {
  client = createClient(baseUrl, token ? { token } : undefined);
}

// Stable facade that always delegates to the current client, so modules can
// keep importing `api` while the auth token changes at runtime.
export const api: TcgApiClient = new Proxy({} as TcgApiClient, {
  get(_target, prop) {
    return client[prop as keyof TcgApiClient];
  },
});
