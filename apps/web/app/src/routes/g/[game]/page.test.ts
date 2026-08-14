import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@testing-library/svelte";
import type { SetSummary } from "@tcg/api-client";
import SetsPage from "./+page.svelte";

const { fetchSets, goto } = vi.hoisted(() => ({
  fetchSets: vi.fn<(game: string, query?: string) => Promise<SetSummary[]>>(),
  goto: vi.fn(),
}));

vi.mock("$app/navigation", () => ({
  goto: (...args: unknown[]) => goto(...args),
}));

vi.mock("@tcg/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tcg/api-client")>();
  return {
    ...actual,
    createClient: () => ({ ...({} as import("@tcg/api-client").TcgApiClient), fetchSets }),
  };
});

function set(key: string, name: string): SetSummary {
  return {
    id: `id-${key}`,
    key,
    language: "en",
    gameId: "id-pokemon",
    name,
    cardCount: 102,
    releaseDate: "1999-01-09",
    cardTotal: 102,
    logoUrl: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}

describe("sets page", () => {
  beforeEach(() => {
    fetchSets.mockReset();
    goto.mockReset();
  });

  it("fetches and renders the sets for the game in the URL", async () => {
    fetchSets.mockResolvedValue([set("base", "Base Set")]);

    render(SetsPage, { props: { data: { game: "pokemon", query: "" }, params: { game: "pokemon" } } });

    const tile = await screen.findByText("Base Set");
    expect(tile.closest("a")?.getAttribute("href")).toBe("/g/pokemon/s/base");
    expect(fetchSets).toHaveBeenCalledWith("pokemon", "");
  });

  it("navigates with a query parameter when a search is submitted", async () => {
    fetchSets.mockResolvedValue([]);

    render(SetsPage, { props: { data: { game: "pokemon", query: "" }, params: { game: "pokemon" } } });

    const input = screen.getByLabelText("Search sets");
    await fireEvent.input(input, { target: { value: "jungle" } });
    const form = input.closest("form");
    expect(form).toBeTruthy();
    await fireEvent.submit(form as HTMLFormElement);

    expect(goto).toHaveBeenCalledWith("/g/pokemon?query=jungle");
  });
});
