// The pokemon-price-updater Lambda: a scheduled job (EventBridge) that
// refreshes variant prices for every Pokémon card listed on TCGplayer.
//
// Flow per run:
//  1. list the game's cards that carry a tcgplayerId,
//  2. fetch current prices from the price feed (PRICE_API_URL) in batches,
//  3. overwrite each card's variant prices in place (no history — that is a
//     catalog invariant, see libs/catalog).
//
// The job is idempotent: re-running it converges on the same state.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
)

// batchSize is how many TCGplayer ids are quoted per feed request.
const batchSize = 100

// Store is the slice of the catalog this job uses.
type Store interface {
	ResolveGame(ctx context.Context, idOrKey string) (catalog.Game, error)
	PricedCards(ctx context.Context, gameID string) ([]catalog.PricedCard, error)
	SetCardPrices(ctx context.Context, gameID string, tcgplayerID int, prices map[string]float64) error
}

// PriceSource quotes current prices by TCGplayer product id:
// id → variant name → price. Ids with no quote are simply absent.
type PriceSource interface {
	Prices(ctx context.Context, tcgplayerIDs []int) (map[int]map[string]float64, error)
}

type Job struct {
	Store   Store
	Source  PriceSource
	GameKey string // catalog game key, e.g. "pokemon"
	Log     *slog.Logger
}

// Result summarizes one run (returned to Lambda for CloudWatch visibility).
type Result struct {
	CardsListed  int `json:"cardsListed"`
	CardsUpdated int `json:"cardsUpdated"`
	CardsFailed  int `json:"cardsFailed"`
}

// Run executes one update pass. Per-card write failures are logged and
// counted, not fatal — one bad card must not abort the whole run. Feed
// failures are fatal: without quotes there is nothing to do.
func (j *Job) Run(ctx context.Context) (Result, error) {
	game, err := j.Store.ResolveGame(ctx, j.GameKey)
	if err != nil {
		return Result{}, fmt.Errorf("resolve game %q: %w", j.GameKey, err)
	}
	cards, err := j.Store.PricedCards(ctx, game.ID)
	if err != nil {
		return Result{}, fmt.Errorf("list priced cards: %w", err)
	}
	res := Result{CardsListed: len(cards)}

	for start := 0; start < len(cards); start += batchSize {
		end := min(start+batchSize, len(cards))
		batch := cards[start:end]

		ids := make([]int, len(batch))
		for i, card := range batch {
			ids[i] = card.TCGPlayerID
		}
		quotes, err := j.Source.Prices(ctx, ids)
		if err != nil {
			return res, fmt.Errorf("price feed: %w", err)
		}
		for _, card := range batch {
			prices, ok := quotes[card.TCGPlayerID]
			if !ok || len(prices) == 0 {
				continue
			}
			if err := j.Store.SetCardPrices(ctx, game.ID, card.TCGPlayerID, prices); err != nil {
				res.CardsFailed++
				j.Log.Error("price write failed", "cardId", card.ID, "tcgplayerId", card.TCGPlayerID, "err", err)
				continue
			}
			res.CardsUpdated++
		}
	}
	if res.CardsFailed > 0 {
		return res, fmt.Errorf("%d of %d price writes failed", res.CardsFailed, res.CardsListed)
	}
	return res, nil
}

// ErrNoSource is returned at startup when PRICE_API_URL is missing.
var ErrNoSource = errors.New("PRICE_API_URL is not set")
