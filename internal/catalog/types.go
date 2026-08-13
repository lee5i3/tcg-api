// Package catalog is the card catalog: games, sets, cards, and their variants
// in PostgreSQL. The service is standalone — data enters exclusively through
// the write API, never from an upstream source.
//
// Data-model rules:
//   - prices live on card_variants and sealed products, overwritten in place;
//     nothing is stored on the card row itself
//   - set sizes are never stored, always counted live from cards
//   - games and sets have a GUID id plus an immutable catalog `key`; cards
//     have a GUID id plus an optional tcgplayer_id, and lookups accept either
package catalog

import "time"

// CardVariant is one printing a card exists in ("Normal", "Holofoil",
// "Reverse Holofoil", "Cold Foil", "Unlimited", "1st Edition", ...) with
// that printing's current price.
type CardVariant struct {
	ID    string   `json:"id"` // GUID
	Name  string   `json:"name"`
	Price *float64 `json:"price"`
}

type Game struct {
	ID        string    `json:"id"`       // GUID
	Key       string    `json:"key"`      // immutable routing key, unique (e.g. "pokemon")
	Language  string    `json:"language"` // ISO 639 alpha-3 ("eng", "jpn"); immutable
	Label     string    `json:"label"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SetSummary struct {
	ID          string    `json:"id"`       // GUID
	Key         string    `json:"key"`      // catalog set id (e.g. "sv3pt5"), immutable
	Language    string    `json:"language"` // ISO 639 alpha-3; immutable
	GameID      string    `json:"gameId"`
	Name        string    `json:"name"`
	CardCount   int       `json:"cardCount"`   // live count, never stored
	ReleaseDate *string   `json:"releaseDate"` // YYYY-MM-DD
	CardTotal   *int      `json:"cardTotal"`   // official printed set size
	LogoURL     *string   `json:"logoUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CardSummary is the list/search shape. The set is referenced by id only —
// set details come from the sets endpoints.
type CardSummary struct {
	ID          string   `json:"id"`          // GUID
	TCGPlayerID *int      `json:"tcgplayerId"` // TCGplayer product id; nil when not listed there
	Language    string    `json:"language"`    // ISO 639 alpha-3; immutable
	Name        string    `json:"name"`
	Number      string    `json:"number"`
	Rarity      *string       `json:"rarity"`
	SetID       string        `json:"setId"` // set GUID
	Image       *string       `json:"image"`
	ImageLarge  *string       `json:"imageLarge"`
	Variants    []CardVariant `json:"variants"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type CardImages struct {
	Small *string `json:"small"`
	Large *string `json:"large"`
}

type CardDetail struct {
	ID          string     `json:"id"`
	TCGPlayerID *int       `json:"tcgplayerId"` // TCGplayer product id; nil when not listed there
	Language    string     `json:"language"`    // ISO 639 alpha-3; immutable
	GameID      string     `json:"gameId"`      // games.id GUID
	Name      string     `json:"name"`
	Number    *string       `json:"number"`
	Rarity    *string       `json:"rarity"`
	Set       string        `json:"set"` // set GUID — details via the sets endpoints
	Images    CardImages    `json:"images"`
	Variants  []CardVariant `json:"variants"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// CardInput is the write shape for creating/updating a card.
// Language applies on create only; it is immutable afterwards.
type CardInput struct {
	TCGPlayerID *int    `json:"tcgplayerId"` // TCGplayer product id
	Language    *string `json:"language"`    // ISO 639 alpha-3; nil means "eng"
	SetID       string  `json:"setId"`       // set GUID or key
	Name        string  `json:"name"`
	Number      *string `json:"number"`
	Rarity      *string `json:"rarity"`
	ImageSmall  *string `json:"imageSmall"`
	ImageLarge  *string `json:"imageLarge"`
}

// SetInput is the write shape for creating/updating a set.
// Language applies on create only; it is immutable afterwards.
type SetInput struct {
	Key         string  `json:"key"`      // required on create; immutable
	Language    *string `json:"language"` // ISO 639 alpha-3; nil means "eng"
	Name        string  `json:"name"`
	ReleaseDate *string `json:"releaseDate"` // YYYY-MM-DD (YYYY/MM/DD accepted)
	CardTotal   *int    `json:"cardTotal"` // official printed set size
	LogoURL     *string `json:"logoUrl"`
}

