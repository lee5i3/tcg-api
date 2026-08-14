package catalog

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// newTestCatalog wires a store to the in-memory fake with deterministic
// ids (id-1, id-2, ...) and a ticking clock (1s per call).
func newTestCatalog() (*Catalog, *fakeDynamo) {
	fake := newFakeDynamo()
	c := New(fake, "test-table")
	ids := 0
	c.newID = func() string {
		ids++
		return fmt.Sprintf("id-%d", ids)
	}
	tick := 0
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time {
		tick++
		return base.Add(time.Duration(tick) * time.Second)
	}
	return c, fake
}

func mustAddGame(t *testing.T, c *Catalog, key, label string) Game {
	t.Helper()
	g, err := c.AddGame(context.Background(), key, label, nil)
	if err != nil {
		t.Fatalf("AddGame(%q): %v", key, err)
	}
	return g
}

func mustCreateSet(t *testing.T, c *Catalog, gameID string, in SetInput) string {
	t.Helper()
	id, err := c.CreateSet(context.Background(), gameID, in)
	if err != nil {
		t.Fatalf("CreateSet(%q): %v", in.Key, err)
	}
	return id
}

func mustCreateCard(t *testing.T, c *Catalog, g Game, in CardInput) string {
	t.Helper()
	id, err := c.CreateCard(context.Background(), g, in)
	if err != nil {
		t.Fatalf("CreateCard(%q): %v", in.Name, err)
	}
	return id
}

func ptr[T any](v T) *T { return &v }

// ---------------------------------------------------------------------------
// Games
// ---------------------------------------------------------------------------

func TestAddGameAndList(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()

	g1 := mustAddGame(t, c, "pokemon", "Pokémon")
	g2 := mustAddGame(t, c, "magic", "Magic: The Gathering")

	if g1.Language != "eng" {
		t.Errorf("default language = %q, want eng", g1.Language)
	}
	games, err := c.Games(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 {
		t.Fatalf("got %d games, want 2", len(games))
	}
	// Creation order, oldest first.
	if games[0].ID != g1.ID || games[1].ID != g2.ID {
		t.Errorf("games out of creation order: %v", games)
	}
}

func TestAddGameValidation(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()

	for _, tc := range []struct{ key, label string }{
		{"P!", "bad key"},
		{"x", "too short"},
		{"pokemon", "   "},
	} {
		if _, err := c.AddGame(ctx, tc.key, tc.label, nil); !errors.Is(err, ErrInvalid) {
			t.Errorf("AddGame(%q, %q) err = %v, want ErrInvalid", tc.key, tc.label, err)
		}
	}
	if _, err := c.AddGame(ctx, "jp", "Japanese cards", ptr("EN GLISH")); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad language err = %v, want ErrInvalid", err)
	}
}

func TestAddGameConflict(t *testing.T) {
	c, _ := newTestCatalog()
	mustAddGame(t, c, "pokemon", "Pokémon")
	if _, err := c.AddGame(context.Background(), "pokemon", "Again", nil); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate key err = %v, want ErrConflict", err)
	}
}

func TestResolveGame(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")

	byID, err := c.ResolveGame(ctx, g.ID)
	if err != nil || byID.Key != "pokemon" {
		t.Errorf("by id: %v, %v", byID, err)
	}
	byKey, err := c.ResolveGame(ctx, "pokemon")
	if err != nil || byKey.ID != g.ID {
		t.Errorf("by key: %v, %v", byKey, err)
	}
	if _, err := c.ResolveGame(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing game err = %v, want ErrNotFound", err)
	}
}

func TestSeed(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	if err := c.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	games, _ := c.Games(ctx)
	if len(games) != 4 {
		t.Fatalf("seeded %d games, want 4", len(games))
	}
	// Second seed is a no-op.
	if err := c.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	games, _ = c.Games(ctx)
	if len(games) != 4 {
		t.Errorf("re-seed changed count to %d", len(games))
	}
}

// ---------------------------------------------------------------------------
// Sets
// ---------------------------------------------------------------------------

func TestCreateAndListSets(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")

	mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151", ReleaseDate: ptr("2023/09/22"), CardTotal: ptr(165)})
	mustCreateSet(t, c, g.ID, SetInput{Key: "base1", Name: "Base Set", ReleaseDate: ptr("1999-01-09")})
	mustCreateSet(t, c, g.ID, SetInput{Key: "promo", Name: "Promos"}) // no release date

	sets, err := c.ListSets(ctx, g.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 3 {
		t.Fatalf("got %d sets, want 3", len(sets))
	}
	// Newest release first, undated last.
	if sets[0].Key != "sv3pt5" || sets[1].Key != "base1" || sets[2].Key != "promo" {
		t.Errorf("order = %s, %s, %s", sets[0].Key, sets[1].Key, sets[2].Key)
	}
	// The slash date was normalized.
	if sets[0].ReleaseDate == nil || *sets[0].ReleaseDate != "2023-09-22" {
		t.Errorf("releaseDate = %v, want 2023-09-22", sets[0].ReleaseDate)
	}

	filtered, err := c.ListSets(ctx, g.ID, "bAsE")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].Key != "base1" {
		t.Errorf("filter = %v", filtered)
	}
}

func TestCreateSetConflictAndValidation(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})

	if _, err := c.CreateSet(ctx, g.ID, SetInput{Key: "sv3pt5", Name: "Duplicate"}); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate set err = %v, want ErrConflict", err)
	}
	if _, err := c.CreateSet(ctx, g.ID, SetInput{Key: "  ", Name: "Blank"}); !errors.Is(err, ErrInvalid) {
		t.Errorf("blank key err = %v, want ErrInvalid", err)
	}
	if _, err := c.CreateSet(ctx, g.ID, SetInput{Key: "x", Name: "Bad date", ReleaseDate: ptr("Sept 22")}); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad date err = %v, want ErrInvalid", err)
	}
}

func TestUpdateSet(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151", CardTotal: ptr(165)})

	// Update by key; cardTotal cleared, name changed.
	err := c.UpdateSet(ctx, g.ID, "sv3pt5", SetInput{Name: "151 (renamed)", ReleaseDate: ptr("2023-09-22")})
	if err != nil {
		t.Fatal(err)
	}
	sets, _ := c.ListSets(ctx, g.ID, "")
	if sets[0].Name != "151 (renamed)" || sets[0].CardTotal != nil || *sets[0].ReleaseDate != "2023-09-22" {
		t.Errorf("after update: %+v", sets[0])
	}
	if err := c.UpdateSet(ctx, g.ID, "missing", SetInput{Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing set err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSetCascades(t *testing.T) {
	c, fake := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	for i := range 30 { // more than one BatchWriteItem page
		mustCreateCard(t, c, g, CardInput{SetID: setID, Name: fmt.Sprintf("Card %d", i)})
	}

	n, err := c.DeleteSet(ctx, g.ID, "sv3pt5")
	if err != nil {
		t.Fatal(err)
	}
	if n != 30 {
		t.Errorf("cards deleted = %d, want 30", n)
	}
	if _, err := c.SetCards(ctx, g.ID, setID); !errors.Is(err, ErrNotFound) {
		t.Errorf("set still resolvable after delete: %v", err)
	}
	// The key guard went with it, so the key can be reused.
	if _, err := c.CreateSet(ctx, g.ID, SetInput{Key: "sv3pt5", Name: "Reborn"}); err != nil {
		t.Errorf("re-creating deleted set key: %v", err)
	}
	// Nothing left but game + guard + new set + new guard.
	if got := len(fake.items); got != 4 {
		t.Errorf("%d items left in table, want 4", got)
	}
}

// ---------------------------------------------------------------------------
// Cards
// ---------------------------------------------------------------------------

func TestCreateCardMaintainsCount(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})

	// Create by set key (not GUID) to exercise resolution.
	mustCreateCard(t, c, g, CardInput{SetID: "sv3pt5", Name: "Bulbasaur", Number: ptr("1")})
	mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos", Number: ptr("145"), TCGPlayerID: ptr(501773)})

	sets, _ := c.ListSets(ctx, g.ID, "")
	if sets[0].CardCount != 2 {
		t.Errorf("cardCount = %d, want 2", sets[0].CardCount)
	}
	if _, err := c.CreateCard(ctx, g, CardInput{SetID: "missing", Name: "Lost"}); !errors.Is(err, ErrInvalid) {
		t.Errorf("card in missing set err = %v, want ErrInvalid", err)
	}
}

func TestSetCardsCollectorOrder(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	for _, n := range []string{"10", "2", "GG01", "1a", "1"} {
		mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Card " + n, Number: ptr(n)})
	}

	cards, err := c.SetCards(ctx, g.ID, "sv3pt5")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, card := range cards {
		got = append(got, card.Number)
	}
	want := []string{"GG01", "1", "1a", "2", "10"} // no digits → 0, then numeric, then string
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestSearchCards(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	other := mustAddGame(t, c, "magic", "Magic")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	otherSet := mustCreateSet(t, c, other.ID, SetInput{Key: "mh3", Name: "Modern Horizons 3"})

	mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos"})
	mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos ex"})
	mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Pikachu"})
	mustCreateCard(t, c, other, CardInput{SetID: otherSet, Name: "Zapdos Impostor"})

	// Case-insensitive substring, scoped to the game, alphabetical.
	cards, err := c.SearchCards(ctx, g.ID, "  ZAPD ")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 2 || cards[0].Name != "Zapdos" || cards[1].Name != "Zapdos ex" {
		t.Errorf("search = %+v", cards)
	}
	// Blank query returns nothing.
	if cards, _ := c.SearchCards(ctx, g.ID, "   "); len(cards) != 0 {
		t.Errorf("blank query returned %d cards", len(cards))
	}
}

func TestSearchCardsLimit(t *testing.T) {
	c, _ := newTestCatalog()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	for i := range 60 {
		mustCreateCard(t, c, g, CardInput{SetID: setID, Name: fmt.Sprintf("Pikachu %02d", i)})
	}
	cards, err := c.SearchCards(context.Background(), g.ID, "pikachu")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != searchLimit {
		t.Errorf("got %d cards, want the %d cap", len(cards), searchLimit)
	}
}

func TestGetCardByIDAndTCGPlayerID(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	other := mustAddGame(t, c, "magic", "Magic")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	cardID := mustCreateCard(t, c, g, CardInput{
		SetID: setID, Name: "Zapdos", Number: ptr("145"), TCGPlayerID: ptr(501773),
	})

	byID, err := c.GetCard(ctx, g.ID, cardID)
	if err != nil || byID.Name != "Zapdos" || byID.Set != setID {
		t.Errorf("by id: %+v, %v", byID, err)
	}
	if byID.Variants == nil || len(byID.Variants) != 0 {
		t.Errorf("variants should be empty non-nil, got %#v", byID.Variants)
	}
	byTCGP, err := c.GetCard(ctx, g.ID, "501773")
	if err != nil || byTCGP.ID != cardID {
		t.Errorf("by tcgplayer id: %+v, %v", byTCGP, err)
	}
	// A card is invisible from another game.
	if _, err := c.GetCard(ctx, other.ID, cardID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-game get err = %v, want ErrNotFound", err)
	}
	if _, err := c.GetCard(ctx, g.ID, "id-999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing card err = %v, want ErrNotFound", err)
	}
}

func TestUpdateCardInPlace(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	cardID := mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos", Number: ptr("145")})

	err := c.UpdateCard(ctx, g, cardID, CardInput{
		SetID: setID, Name: "Zapdos ex", Number: ptr("145a"), Rarity: ptr("Rare"),
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := c.GetCard(ctx, g.ID, cardID)
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != "Zapdos ex" || *card.Number != "145a" || *card.Rarity != "Rare" {
		t.Errorf("after update: %+v", card)
	}
	// Search sees the new name.
	found, _ := c.SearchCards(ctx, g.ID, "zapdos ex")
	if len(found) != 1 {
		t.Errorf("search after rename found %d", len(found))
	}
}

func TestUpdateCardMovesSets(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setA := mustCreateSet(t, c, g.ID, SetInput{Key: "set-a", Name: "A"})
	setB := mustCreateSet(t, c, g.ID, SetInput{Key: "set-b", Name: "B"})
	cardID := mustCreateCard(t, c, g, CardInput{SetID: setA, Name: "Zapdos"})

	if err := c.UpdateCard(ctx, g, cardID, CardInput{SetID: "set-b", Name: "Zapdos"}); err != nil {
		t.Fatal(err)
	}
	card, _ := c.GetCard(ctx, g.ID, cardID)
	if card.Set != setB {
		t.Errorf("card set = %s, want %s", card.Set, setB)
	}
	sets, _ := c.ListSets(ctx, g.ID, "")
	for _, s := range sets {
		want := map[string]int{setA: 0, setB: 1}[s.ID]
		if s.CardCount != want {
			t.Errorf("set %s cardCount = %d, want %d", s.Key, s.CardCount, want)
		}
	}
}

func TestPricedCards(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos", TCGPlayerID: ptr(501773)})
	mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Promo card"}) // no TCGplayer id

	cards, err := c.PricedCards(ctx, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].TCGPlayerID != 501773 {
		t.Errorf("priced cards = %+v", cards)
	}
}

func TestSetCardPrices(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	cardID := mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos", TCGPlayerID: ptr(501773)})

	// First write creates variants.
	if err := c.SetCardPrices(ctx, g.ID, 501773, map[string]float64{"Normal": 1.42, "Holofoil": 5.00}); err != nil {
		t.Fatal(err)
	}
	card, _ := c.GetCard(ctx, g.ID, cardID)
	if len(card.Variants) != 2 {
		t.Fatalf("variants = %+v", card.Variants)
	}
	// Alphabetical: Holofoil, Normal.
	if card.Variants[0].Name != "Holofoil" || *card.Variants[0].Price != 5.00 {
		t.Errorf("variant[0] = %+v", card.Variants[0])
	}

	// Second write overwrites in place (case-insensitive name match), no dupes.
	if err := c.SetCardPrices(ctx, g.ID, 501773, map[string]float64{"normal": 1.10}); err != nil {
		t.Fatal(err)
	}
	card, _ = c.GetCard(ctx, g.ID, cardID)
	if len(card.Variants) != 2 {
		t.Fatalf("variants after update = %+v", card.Variants)
	}
	if *card.Variants[1].Price != 1.10 {
		t.Errorf("Normal price = %v, want 1.10", *card.Variants[1].Price)
	}

	// Unknown card errors; empty prices is a no-op.
	if err := c.SetCardPrices(ctx, g.ID, 999, map[string]float64{"Normal": 1}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown card err = %v, want ErrNotFound", err)
	}
	if err := c.SetCardPrices(ctx, g.ID, 999, nil); err != nil {
		t.Errorf("empty prices err = %v, want nil", err)
	}
}

func TestDeleteCard(t *testing.T) {
	c, _ := newTestCatalog()
	ctx := context.Background()
	g := mustAddGame(t, c, "pokemon", "Pokémon")
	setID := mustCreateSet(t, c, g.ID, SetInput{Key: "sv3pt5", Name: "151"})
	cardID := mustCreateCard(t, c, g, CardInput{SetID: setID, Name: "Zapdos", TCGPlayerID: ptr(42)})

	// Delete by TCGplayer id.
	if err := c.DeleteCard(ctx, g.ID, "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetCard(ctx, g.ID, cardID); !errors.Is(err, ErrNotFound) {
		t.Errorf("card still there: %v", err)
	}
	sets, _ := c.ListSets(ctx, g.ID, "")
	if sets[0].CardCount != 0 {
		t.Errorf("cardCount = %d, want 0", sets[0].CardCount)
	}
	if err := c.DeleteCard(ctx, g.ID, "42"); !errors.Is(err, ErrNotFound) {
		t.Errorf("double delete err = %v, want ErrNotFound", err)
	}
}
