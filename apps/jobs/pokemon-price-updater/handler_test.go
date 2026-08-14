package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
)

type fakeStore struct {
	cards    []catalog.PricedCard
	written  map[int]map[string]float64
	failIDs  map[int]bool
	gameSeen string
}

func (f *fakeStore) ResolveGame(_ context.Context, idOrKey string) (catalog.Game, error) {
	f.gameSeen = idOrKey
	if idOrKey != "pokemon" {
		return catalog.Game{}, fmt.Errorf("%w: game %q", catalog.ErrNotFound, idOrKey)
	}
	return catalog.Game{ID: "game-1", Key: "pokemon"}, nil
}

func (f *fakeStore) PricedCards(_ context.Context, gameID string) ([]catalog.PricedCard, error) {
	return f.cards, nil
}

func (f *fakeStore) SetCardPrices(_ context.Context, _ string, tcgplayerID int, prices map[string]float64) error {
	if f.failIDs[tcgplayerID] {
		return errors.New("write failed")
	}
	if f.written == nil {
		f.written = map[int]map[string]float64{}
	}
	f.written[tcgplayerID] = prices
	return nil
}

type fakeSource struct {
	quotes  map[int]map[string]float64
	err     error
	batches [][]int
}

func (f *fakeSource) Prices(_ context.Context, ids []int) (map[int]map[string]float64, error) {
	f.batches = append(f.batches, ids)
	if f.err != nil {
		return nil, f.err
	}
	out := map[int]map[string]float64{}
	for _, id := range ids {
		if q, ok := f.quotes[id]; ok {
			out[id] = q
		}
	}
	return out, nil
}

func testJob(store *fakeStore, source *fakeSource) *Job {
	return &Job{Store: store, Source: source, GameKey: "pokemon", Log: slog.New(slog.DiscardHandler)}
}

func TestRunUpdatesQuotedCards(t *testing.T) {
	store := &fakeStore{cards: []catalog.PricedCard{
		{ID: "c1", TCGPlayerID: 100},
		{ID: "c2", TCGPlayerID: 200}, // no quote from the feed
		{ID: "c3", TCGPlayerID: 300},
	}}
	source := &fakeSource{quotes: map[int]map[string]float64{
		100: {"Normal": 1.42},
		300: {"Holofoil": 9.99, "Normal": 2.00},
	}}
	res, err := testJob(store, source).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.CardsListed != 3 || res.CardsUpdated != 2 || res.CardsFailed != 0 {
		t.Errorf("result = %+v", res)
	}
	if store.written[300]["Holofoil"] != 9.99 {
		t.Errorf("written = %+v", store.written)
	}
	if _, ok := store.written[200]; ok {
		t.Error("card without quote must not be written")
	}
}

func TestRunBatchesFeedRequests(t *testing.T) {
	store := &fakeStore{}
	for i := range 250 {
		store.cards = append(store.cards, catalog.PricedCard{ID: fmt.Sprintf("c%d", i), TCGPlayerID: 1000 + i})
	}
	source := &fakeSource{}
	if _, err := testJob(store, source).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(source.batches) != 3 || len(source.batches[0]) != 100 || len(source.batches[2]) != 50 {
		sizes := []int{}
		for _, b := range source.batches {
			sizes = append(sizes, len(b))
		}
		t.Errorf("batch sizes = %v, want [100 100 50]", sizes)
	}
}

func TestRunCountsWriteFailures(t *testing.T) {
	store := &fakeStore{
		cards:   []catalog.PricedCard{{ID: "c1", TCGPlayerID: 100}, {ID: "c2", TCGPlayerID: 200}},
		failIDs: map[int]bool{200: true},
	}
	source := &fakeSource{quotes: map[int]map[string]float64{
		100: {"Normal": 1.0},
		200: {"Normal": 2.0},
	}}
	res, err := testJob(store, source).Run(context.Background())
	if err == nil {
		t.Fatal("want error when writes fail")
	}
	if res.CardsUpdated != 1 || res.CardsFailed != 1 {
		t.Errorf("result = %+v", res)
	}
}

func TestRunFeedFailureIsFatal(t *testing.T) {
	store := &fakeStore{cards: []catalog.PricedCard{{ID: "c1", TCGPlayerID: 100}}}
	source := &fakeSource{err: errors.New("feed down")}
	if _, err := testJob(store, source).Run(context.Background()); err == nil {
		t.Fatal("want error when the feed fails")
	}
}

func TestHTTPPriceSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("ids"); got != "100,200" {
			t.Errorf("ids param = %q", got)
		}
		fmt.Fprint(w, `{"prices": {"100": {"Normal": 1.42}, "junk": {"X": 1}}}`)
	}))
	defer srv.Close()

	quotes, err := NewHTTPPriceSource(srv.URL).Prices(context.Background(), []int{100, 200})
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 1 || quotes[100]["Normal"] != 1.42 {
		t.Errorf("quotes = %+v", quotes)
	}
}

func TestHTTPPriceSourceErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := NewHTTPPriceSource(srv.URL).Prices(context.Background(), []int{1}); err == nil {
		t.Fatal("want error on non-200 feed response")
	}
}
