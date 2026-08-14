import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/svelte";
import type { Game } from "@tcg/api-client";
import GamesPage from "./+page.svelte";

const { fetchGames, createGame } = vi.hoisted(() => ({
  fetchGames: vi.fn<() => Promise<Game[]>>(),
  createGame: vi.fn<(input: unknown) => Promise<Game>>(),
}));

vi.mock("@tcg/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tcg/api-client")>();
  return {
    ...actual,
    createClient: () =>
      ({ fetchGames, createGame }) as unknown as import("@tcg/api-client").TcgApiClient,
  };
});

function game(key: string, label: string): Game {
  return { id: `id-${key}`, key, language: "en", label, updatedAt: "2026-01-01T00:00:00Z" };
}

describe("admin games page", () => {
  beforeEach(() => {
    fetchGames.mockReset();
    createGame.mockReset();
  });

  it("loads and lists games", async () => {
    fetchGames.mockResolvedValue([game("pokemon", "Pokémon")]);

    render(GamesPage);

    expect(await screen.findByText("Pokémon")).toBeTruthy();
    expect(fetchGames).toHaveBeenCalledTimes(1);
  });

  it("submits the create-game form and reloads the list", async () => {
    fetchGames.mockResolvedValue([]);
    createGame.mockResolvedValue(game("lorcana", "Lorcana"));

    render(GamesPage);
    await screen.findByText("No games yet.");

    await fireEvent.input(screen.getByLabelText("Key"), { target: { value: " lorcana " } });
    await fireEvent.input(screen.getByLabelText("Label"), { target: { value: "Lorcana" } });
    const form = screen.getByLabelText("Create game");
    await fireEvent.submit(form);

    await vi.waitFor(() => {
      expect(createGame).toHaveBeenCalledWith({ key: "lorcana", label: "Lorcana" });
    });
    // Initial load + reload after the successful create.
    await vi.waitFor(() => {
      expect(fetchGames).toHaveBeenCalledTimes(2);
    });
  });

  it("shows an inline ApiError when the create fails (e.g. 409 conflict)", async () => {
    const { ApiError } = await import("@tcg/api-client");
    fetchGames.mockResolvedValue([]);
    createGame.mockRejectedValue(new ApiError(409, "Request to /v1/games failed with status 409"));

    render(GamesPage);
    await screen.findByText("No games yet.");

    await fireEvent.input(screen.getByLabelText("Key"), { target: { value: "pokemon" } });
    await fireEvent.input(screen.getByLabelText("Label"), { target: { value: "Pokémon" } });
    await fireEvent.submit(screen.getByLabelText("Create game"));

    expect(await screen.findByRole("alert")).toHaveProperty(
      "textContent",
      "Request to /v1/games failed with status 409",
    );
  });
});
