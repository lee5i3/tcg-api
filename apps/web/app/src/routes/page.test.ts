import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/svelte";
import type { Game } from "@tcg/api-client";
import GamesPage from "./+page.svelte";

const { fetchGames } = vi.hoisted(() => ({
  fetchGames: vi.fn<() => Promise<Game[]>>(),
}));

vi.mock("@tcg/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tcg/api-client")>();
  return {
    ...actual,
    createClient: () => ({ ...({} as import("@tcg/api-client").TcgApiClient), fetchGames }),
  };
});

function game(key: string, label: string): Game {
  return { id: `id-${key}`, key, language: "en", label, updatedAt: "2026-01-01T00:00:00Z" };
}

describe("games page", () => {
  beforeEach(() => {
    fetchGames.mockReset();
  });

  it("shows a loading state, then the games from the API", async () => {
    fetchGames.mockResolvedValue([game("pokemon", "Pokémon"), game("mtg", "Magic")]);

    render(GamesPage);

    expect(screen.getByText("Loading…")).toBeTruthy();

    const pokemon = await screen.findByText("Pokémon");
    expect(pokemon.closest("a")?.getAttribute("href")).toBe("/g/pokemon");
    expect(screen.getByText("Magic")).toBeTruthy();
    expect(fetchGames).toHaveBeenCalledTimes(1);
  });

  it("shows an empty state when the API returns no games", async () => {
    fetchGames.mockResolvedValue([]);

    render(GamesPage);

    expect(await screen.findByText("No games available.")).toBeTruthy();
  });

  it("shows an error state when the API call fails", async () => {
    fetchGames.mockRejectedValue(new Error("boom"));

    render(GamesPage);

    expect(await screen.findByText("Something went wrong: boom")).toBeTruthy();
  });
});
