export interface Game {
  id: string;
  key: string;
  language: string;
  label: string;
  updatedAt: string;
}

export interface SetSummary {
  id: string;
  key: string;
  language: string;
  gameId: string;
  name: string;
  cardCount: number;
  releaseDate: string | null;
  cardTotal: number | null;
  logoUrl: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CardVariant {
  id: string;
  name: string;
  price: number | null;
}

export interface CardSummary {
  id: string;
  tcgplayerId: number | null;
  language: string;
  name: string;
  number: string;
  rarity: string | null;
  setId: string;
  image: string | null;
  imageLarge: string | null;
  variants: CardVariant[];
  createdAt: string;
  updatedAt: string;
}

export interface CardImages {
  small: string | null;
  large: string | null;
}

export interface CardDetail {
  id: string;
  tcgplayerId: number | null;
  language: string;
  gameId: string;
  name: string;
  number: string | null;
  rarity: string | null;
  set: SetSummary;
  images: CardImages;
  variants: CardVariant[];
  createdAt: string;
  updatedAt: string;
}

export interface User {
  id: string;
  email: string;
  name: string | null;
  createdAt: string;
}

/** A signed-in user session: the JWT plus the user it belongs to. */
export interface AuthSession {
  token: string;
  user: User;
}

/** A social sign-in provider from GET /v1/auth/providers. */
export interface AuthProvider {
  id: string;
  label: string;
}

/** Body for POST /v1/auth/register. */
export interface RegisterInput {
  email: string;
  password: string;
  name?: string;
}

/** Body for POST /v1/auth/login. */
export interface LoginInput {
  email: string;
  password: string;
}

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/** Body for POST /v1/games. */
export interface GameInput {
  key: string;
  label: string;
  language?: string;
}

/**
 * Body for POST /v1/games/{game}/sets and PUT /v1/games/{game}/sets/{set}
 * (on update, `key` and `language` are ignored by the API).
 */
export interface SetInput {
  key: string;
  name: string;
  language?: string;
  releaseDate?: string;
  cardTotal?: number;
  logoUrl?: string;
}

/**
 * Body for POST /v1/games/{game}/cards and PUT /v1/games/{game}/cards/{card}.
 */
export interface CardInput {
  setId: string;
  name: string;
  number?: string;
  rarity?: string;
  language?: string;
  tcgplayerId?: number;
  imageSmall?: string;
  imageLarge?: string;
}

export interface ClientOptions {
  /** Bearer token sent as `Authorization: Bearer <token>` on every request. */
  token?: string;
}

export interface TcgApiClient {
  fetchGames(): Promise<Game[]>;
  fetchSets(game: string, query?: string): Promise<SetSummary[]>;
  fetchSetCards(game: string, set: string): Promise<CardSummary[]>;
  searchCards(game: string, query?: string): Promise<CardSummary[]>;
  fetchCard(game: string, card: string): Promise<CardDetail>;
  /** POST /v1/auth/check — true on 2xx, false on 401, throws ApiError otherwise. */
  authCheck(): Promise<boolean>;
  /**
   * GET /v1/auth/providers — the social sign-in providers the API is
   * configured for. May be empty when no provider is set up.
   */
  providers(): Promise<AuthProvider[]>;
  /**
   * POST /v1/auth/register — creates a user account and returns its session.
   * Throws ApiError on 400 (invalid email / short password) and 409
   * (email already taken), with the server's error message when available.
   */
  register(input: RegisterInput): Promise<AuthSession>;
  /**
   * POST /v1/auth/login — throws ApiError with status 401 on bad credentials.
   */
  login(input: LoginInput): Promise<AuthSession>;
  /**
   * GET /v1/auth/me — returns the user for the client's bearer token.
   * Throws ApiError with status 401 when the token is missing/invalid/expired.
   */
  me(): Promise<User>;
  createGame(input: GameInput): Promise<Game>;
  createSet(game: string, input: SetInput): Promise<string>;
  updateSet(game: string, set: string, input: SetInput): Promise<void>;
  /** Resolves with the number of cards deleted along with the set. */
  deleteSet(game: string, set: string): Promise<number>;
  createCard(game: string, input: CardInput): Promise<string>;
  updateCard(game: string, card: string, input: CardInput): Promise<void>;
  deleteCard(game: string, card: string): Promise<void>;
}

/**
 * Creates a TCG catalog API client.
 *
 * @param baseUrl Base URL of the API (trailing slashes are stripped).
 *                Pass an empty string for same-origin requests
 *                (relative `/v1/...` URLs).
 * @param options Optional settings; set `token` to authenticate write
 *                requests with `Authorization: Bearer <token>`.
 */
export function createClient(baseUrl: string, options?: ClientOptions): TcgApiClient {
  const base = baseUrl.replace(/\/+$/, "");
  const token = options?.token;

  async function send(
    method: string,
    path: string,
    query?: Record<string, string>,
    body?: unknown,
  ): Promise<Response> {
    let url = `${base}${path}`;
    if (query) {
      const params = new URLSearchParams();
      for (const [key, value] of Object.entries(query)) {
        if (value !== "") {
          params.set(key, value);
        }
      }
      const qs = params.toString();
      if (qs) {
        url += `?${qs}`;
      }
    }

    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const init: RequestInit = { method, headers };
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }

    return fetch(url, init);
  }

  async function request<T>(
    method: string,
    path: string,
    query?: Record<string, string>,
    body?: unknown,
  ): Promise<T> {
    const response = await send(method, path, query, body);
    if (!response.ok) {
      // Prefer the API's own error message ({"error": "..."} bodies).
      let message = `Request to ${path} failed with status ${response.status}`;
      try {
        const errorBody = (await response.json()) as { error?: unknown };
        if (typeof errorBody?.error === "string" && errorBody.error !== "") {
          message = errorBody.error;
        }
      } catch {
        // Non-JSON error body — keep the generic message.
      }
      throw new ApiError(response.status, message);
    }
    if (response.status === 204) {
      return undefined as T;
    }
    return (await response.json()) as T;
  }

  return {
    async fetchGames(): Promise<Game[]> {
      const body = await request<{ games: Game[] }>("GET", "/v1/games");
      return body.games;
    },

    async fetchSets(game: string, query = ""): Promise<SetSummary[]> {
      const body = await request<{ sets: SetSummary[] }>(
        "GET",
        `/v1/games/${encodeURIComponent(game)}/sets`,
        { query },
      );
      return body.sets;
    },

    async fetchSetCards(game: string, set: string): Promise<CardSummary[]> {
      const body = await request<{ cards: CardSummary[] }>(
        "GET",
        `/v1/games/${encodeURIComponent(game)}/sets/${encodeURIComponent(set)}/cards`,
      );
      return body.cards;
    },

    async searchCards(game: string, query = ""): Promise<CardSummary[]> {
      const body = await request<{ cards: CardSummary[] }>(
        "GET",
        `/v1/games/${encodeURIComponent(game)}/cards`,
        { query },
      );
      return body.cards;
    },

    async fetchCard(game: string, card: string): Promise<CardDetail> {
      const body = await request<{ card: CardDetail }>(
        "GET",
        `/v1/games/${encodeURIComponent(game)}/cards/${encodeURIComponent(card)}`,
      );
      return body.card;
    },

    async authCheck(): Promise<boolean> {
      const response = await send("POST", "/v1/auth/check");
      if (response.ok) {
        return true;
      }
      if (response.status === 401) {
        return false;
      }
      throw new ApiError(
        response.status,
        `Request to /v1/auth/check failed with status ${response.status}`,
      );
    },

    async providers(): Promise<AuthProvider[]> {
      const body = await request<{ providers: AuthProvider[] }>("GET", "/v1/auth/providers");
      return body.providers;
    },

    async register(input: RegisterInput): Promise<AuthSession> {
      return request<AuthSession>("POST", "/v1/auth/register", undefined, input);
    },

    async login(input: LoginInput): Promise<AuthSession> {
      return request<AuthSession>("POST", "/v1/auth/login", undefined, input);
    },

    async me(): Promise<User> {
      const body = await request<{ user: User }>("GET", "/v1/auth/me");
      return body.user;
    },

    async createGame(input: GameInput): Promise<Game> {
      const body = await request<{ game: Game }>("POST", "/v1/games", undefined, input);
      return body.game;
    },

    async createSet(game: string, input: SetInput): Promise<string> {
      const body = await request<{ id: string }>(
        "POST",
        `/v1/games/${encodeURIComponent(game)}/sets`,
        undefined,
        input,
      );
      return body.id;
    },

    async updateSet(game: string, set: string, input: SetInput): Promise<void> {
      await request<void>(
        "PUT",
        `/v1/games/${encodeURIComponent(game)}/sets/${encodeURIComponent(set)}`,
        undefined,
        input,
      );
    },

    async deleteSet(game: string, set: string): Promise<number> {
      const body = await request<{ cardsDeleted: number }>(
        "DELETE",
        `/v1/games/${encodeURIComponent(game)}/sets/${encodeURIComponent(set)}`,
      );
      return body.cardsDeleted;
    },

    async createCard(game: string, input: CardInput): Promise<string> {
      const body = await request<{ id: string }>(
        "POST",
        `/v1/games/${encodeURIComponent(game)}/cards`,
        undefined,
        input,
      );
      return body.id;
    },

    async updateCard(game: string, card: string, input: CardInput): Promise<void> {
      await request<void>(
        "PUT",
        `/v1/games/${encodeURIComponent(game)}/cards/${encodeURIComponent(card)}`,
        undefined,
        input,
      );
    },

    async deleteCard(game: string, card: string): Promise<void> {
      await request<void>(
        "DELETE",
        `/v1/games/${encodeURIComponent(game)}/cards/${encodeURIComponent(card)}`,
      );
    },
  };
}

export function lowestPrice(variants: CardVariant[]): number | null {
  let min: number | null = null;
  for (const variant of variants) {
    if (variant.price !== null && (min === null || variant.price < min)) {
      min = variant.price;
    }
  }
  return min;
}

export function formatPrice(price: number): string {
  return `$${price.toFixed(2)}`;
}
