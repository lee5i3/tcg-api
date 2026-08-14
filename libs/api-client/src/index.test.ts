import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, createClient, lowestPrice } from "./index";

function okResponse(body: unknown, status = 200): Response {
  return {
    ok: true,
    status,
    statusText: "OK",
    json: async () => body,
  } as unknown as Response;
}

function noContentResponse(): Response {
  return {
    ok: true,
    status: 204,
    statusText: "No Content",
    json: async () => {
      throw new Error("204 has no body");
    },
  } as unknown as Response;
}

function errorResponse(status: number, statusText: string, body: unknown = {}): Response {
  return {
    ok: false,
    status,
    statusText,
    json: async () => body,
  } as unknown as Response;
}

function stubFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function lastInit(fetchMock: ReturnType<typeof vi.fn>): RequestInit {
  const call = fetchMock.mock.calls.at(-1);
  if (!call) {
    throw new Error("fetch was not called");
  }
  return call[1] as RequestInit;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

const client = createClient("http://localhost:3000");
const authedClient = createClient("http://localhost:3000", { token: "s3cret" });

describe("createClient", () => {
  it("strips trailing slashes from the base URL", async () => {
    const fetchMock = stubFetch(okResponse({ games: [] }));

    await createClient("http://localhost:3000///").fetchGames();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("builds relative same-origin URLs when the base URL is empty", async () => {
    const fetchMock = stubFetch(okResponse({ games: [] }));

    await createClient("").fetchGames();
    expect(fetchMock).toHaveBeenCalledWith("/v1/games", expect.objectContaining({ method: "GET" }));
  });
});

describe("bearer token", () => {
  it("sends Authorization: Bearer on every request when a token is set", async () => {
    const fetchMock = stubFetch(okResponse({ games: [] }));

    await authedClient.fetchGames();
    expect(lastInit(fetchMock).headers).toMatchObject({
      Authorization: "Bearer s3cret",
    });
  });

  it("sends the bearer header on write requests too", async () => {
    const fetchMock = stubFetch(okResponse({ id: "s1" }, 201));

    await authedClient.createSet("pokemon", { key: "base", name: "Base" });
    expect(lastInit(fetchMock).headers).toMatchObject({
      Authorization: "Bearer s3cret",
      "Content-Type": "application/json",
    });
  });

  it("omits the Authorization header when no token is set", async () => {
    const fetchMock = stubFetch(okResponse({ games: [] }));

    await client.fetchGames();
    const headers = lastInit(fetchMock).headers as Record<string, string>;
    expect(headers["Authorization"]).toBeUndefined();
  });
});

describe("authCheck", () => {
  it("returns true when the token is valid (204)", async () => {
    const fetchMock = stubFetch(noContentResponse());

    await expect(authedClient.authCheck()).resolves.toBe(true);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/auth/check",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ Authorization: "Bearer s3cret" }),
      }),
    );
  });

  it("returns false on 401", async () => {
    stubFetch(errorResponse(401, "Unauthorized"));

    await expect(authedClient.authCheck()).resolves.toBe(false);
  });

  it("throws an ApiError on other failures", async () => {
    stubFetch(errorResponse(500, "Internal Server Error"));

    const error = await authedClient.authCheck().catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(500);
  });
});

describe("providers", () => {
  it("fetches the configured providers from GET /v1/auth/providers", async () => {
    const providers = [
      { id: "google", label: "Google" },
      { id: "facebook", label: "Facebook" },
    ];
    const fetchMock = stubFetch(okResponse({ providers }));

    await expect(client.providers()).resolves.toEqual(providers);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/auth/providers",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("returns an empty array when no provider is configured", async () => {
    stubFetch(okResponse({ providers: [] }));

    await expect(client.providers()).resolves.toEqual([]);
  });

  it("propagates an ApiError on failure", async () => {
    stubFetch(errorResponse(500, "Internal Server Error"));

    await expect(client.providers()).rejects.toThrowError(ApiError);
  });
});

describe("user auth", () => {
  const user = {
    id: "u1",
    email: "ash@example.com",
    name: "Ash Ketchum",
    createdAt: "2026-01-01T00:00:00Z",
  };
  const session = { token: "jwt-token", user };

  it("registers with POST /v1/auth/register and returns the session", async () => {
    const fetchMock = stubFetch(okResponse(session, 201));

    await expect(
      client.register({ email: "ash@example.com", password: "pikachu123", name: "Ash Ketchum" }),
    ).resolves.toEqual(session);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/auth/register",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          email: "ash@example.com",
          password: "pikachu123",
          name: "Ash Ketchum",
        }),
      }),
    );
    expect(lastInit(fetchMock).headers).toMatchObject({ "Content-Type": "application/json" });
  });

  it("registers without a name", async () => {
    const fetchMock = stubFetch(okResponse(session, 201));

    await client.register({ email: "ash@example.com", password: "pikachu123" });
    expect(lastInit(fetchMock).body).toBe(
      JSON.stringify({ email: "ash@example.com", password: "pikachu123" }),
    );
  });

  it("propagates a 409 ApiError with the server message when the email is taken", async () => {
    stubFetch(errorResponse(409, "Conflict", { error: "email already registered" }));

    const error = await client
      .register({ email: "ash@example.com", password: "pikachu123" })
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(409);
    expect((error as ApiError).message).toBe("email already registered");
  });

  it("propagates a 400 ApiError for invalid registration input", async () => {
    stubFetch(errorResponse(400, "Bad Request", { error: "password must be at least 8 characters" }));

    const error = await client
      .register({ email: "ash@example.com", password: "short" })
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(400);
    expect((error as ApiError).message).toBe("password must be at least 8 characters");
  });

  it("logs in with POST /v1/auth/login and returns the session", async () => {
    const fetchMock = stubFetch(okResponse(session));

    await expect(
      client.login({ email: "ash@example.com", password: "pikachu123" }),
    ).resolves.toEqual(session);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/auth/login",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ email: "ash@example.com", password: "pikachu123" }),
      }),
    );
  });

  it("propagates a 401 ApiError on bad credentials", async () => {
    stubFetch(errorResponse(401, "Unauthorized", { error: "invalid email or password" }));

    const error = await client
      .login({ email: "ash@example.com", password: "wrong" })
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(401);
    expect((error as ApiError).message).toBe("invalid email or password");
  });

  it("fetches the current user with GET /v1/auth/me using the bearer token", async () => {
    const fetchMock = stubFetch(okResponse({ user }));

    await expect(authedClient.me()).resolves.toEqual(user);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/auth/me",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({ Authorization: "Bearer s3cret" }),
      }),
    );
  });

  it("propagates a 401 ApiError from me() when the token is invalid", async () => {
    stubFetch(errorResponse(401, "Unauthorized", { error: "invalid token" }));

    const error = await authedClient.me().catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(401);
  });
});

describe("read endpoints", () => {
  it("fetches games from /v1/games", async () => {
    const games = [
      { id: "g1", key: "pokemon", language: "en", label: "Pokémon", updatedAt: "2026-01-01T00:00:00Z" },
    ];
    const fetchMock = stubFetch(okResponse({ games }));

    await expect(client.fetchGames()).resolves.toEqual(games);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("builds the sets URL with encoded game and query", async () => {
    const fetchMock = stubFetch(okResponse({ sets: [] }));

    await expect(client.fetchSets("my game", "base set")).resolves.toEqual([]);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/my%20game/sets?query=base+set",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("omits an empty query when fetching sets", async () => {
    const fetchMock = stubFetch(okResponse({ sets: [] }));

    await client.fetchSets("pokemon");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/sets",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("fetches the cards of a set", async () => {
    const cards = [{ id: "c1", name: "Charizard" }];
    const fetchMock = stubFetch(okResponse({ cards }));

    await expect(client.fetchSetCards("pokemon", "base-set")).resolves.toEqual(cards);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/sets/base-set/cards",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("searches cards by name", async () => {
    const fetchMock = stubFetch(okResponse({ cards: [] }));

    await client.searchCards("pokemon", "char");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/cards?query=char",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("fetches a card detail by id or key", async () => {
    const card = { id: "c1", name: "Charizard" };
    const fetchMock = stubFetch(okResponse({ card }));

    await expect(
      client.fetchCard("pokemon", "0f7e2b1a-1111-2222-3333-444455556666"),
    ).resolves.toEqual(card);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/cards/0f7e2b1a-1111-2222-3333-444455556666",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("throws an ApiError with the status on non-2xx responses", async () => {
    stubFetch(errorResponse(404, "Not Found"));

    const error = await client.fetchGames().catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(404);
    expect((error as ApiError).message).toContain("404");
  });

  it("throws on server errors too", async () => {
    stubFetch(errorResponse(500, "Internal Server Error"));

    await expect(client.fetchSets("pokemon")).rejects.toThrowError(ApiError);
  });
});

describe("write endpoints", () => {
  it("creates a game with POST /v1/games and returns the game", async () => {
    const game = {
      id: "g1",
      key: "lorcana",
      language: "en",
      label: "Disney Lorcana",
      updatedAt: "2026-01-01T00:00:00Z",
    };
    const fetchMock = stubFetch(okResponse({ game }, 201));

    await expect(
      authedClient.createGame({ key: "lorcana", label: "Disney Lorcana" }),
    ).resolves.toEqual(game);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ key: "lorcana", label: "Disney Lorcana" }),
      }),
    );
  });

  it("creates a set with POST and returns the new id", async () => {
    const fetchMock = stubFetch(okResponse({ id: "set-1" }, 201));

    await expect(
      authedClient.createSet("pokemon", { key: "base", name: "Base Set", cardTotal: 102 }),
    ).resolves.toBe("set-1");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/sets",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ key: "base", name: "Base Set", cardTotal: 102 }),
      }),
    );
  });

  it("updates a set with PUT and resolves on 204", async () => {
    const fetchMock = stubFetch(noContentResponse());

    await expect(
      authedClient.updateSet("pokemon", "base-set", { key: "base", name: "Base Set" }),
    ).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/sets/base-set",
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("deletes a set and returns cardsDeleted", async () => {
    const fetchMock = stubFetch(okResponse({ cardsDeleted: 42 }));

    await expect(authedClient.deleteSet("pokemon", "base-set")).resolves.toBe(42);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/sets/base-set",
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("creates a card with POST and returns the new id", async () => {
    const fetchMock = stubFetch(okResponse({ id: "card-1" }, 201));

    await expect(
      authedClient.createCard("pokemon", { setId: "set-1", name: "Charizard", number: "4" }),
    ).resolves.toBe("card-1");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/cards",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ setId: "set-1", name: "Charizard", number: "4" }),
      }),
    );
  });

  it("updates a card with PUT and resolves on 204", async () => {
    const fetchMock = stubFetch(noContentResponse());

    await expect(
      authedClient.updateCard("pokemon", "card-1", { setId: "set-1", name: "Charizard" }),
    ).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/cards/card-1",
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("deletes a card and resolves on 204", async () => {
    const fetchMock = stubFetch(noContentResponse());

    await expect(authedClient.deleteCard("pokemon", "card-1")).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/v1/games/pokemon/cards/card-1",
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("throws an ApiError on write failures such as 409 duplicates", async () => {
    stubFetch(errorResponse(409, "Conflict"));

    const error = await authedClient
      .createGame({ key: "pokemon", label: "Pokémon" })
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(409);
  });
});

describe("lowestPrice", () => {
  it("returns the cheapest non-null variant price", () => {
    expect(
      lowestPrice([
        { id: "v1", name: "Normal", price: 4.5 },
        { id: "v2", name: "Holofoil", price: 1.25 },
        { id: "v3", name: "Reverse", price: null },
      ]),
    ).toBe(1.25);
  });

  it("returns null when no variant has a price", () => {
    expect(lowestPrice([{ id: "v1", name: "Normal", price: null }])).toBeNull();
    expect(lowestPrice([])).toBeNull();
  });
});
