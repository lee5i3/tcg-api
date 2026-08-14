package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

type fakeStore struct {
	games []catalog.Game
	added []string
	err   error
}

func (f *fakeStore) Games(context.Context) ([]catalog.Game, error) {
	return f.games, f.err
}

func (f *fakeStore) AddGame(_ context.Context, key, label string, _ *string) (catalog.Game, error) {
	if f.err != nil {
		return catalog.Game{}, f.err
	}
	f.added = append(f.added, key)
	return catalog.Game{ID: "id-1", Key: key, Language: "eng", Label: label}, nil
}

func TestListGames(t *testing.T) {
	api := &API{Store: &fakeStore{games: []catalog.Game{{ID: "g1", Key: "pokemon", Label: "Pokémon"}}}}
	res, err := api.Handler()(context.Background(), httpapi.Request{RouteKey: "GET /v1/games"})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var body struct {
		Games []catalog.Game `json:"games"`
	}
	if err := json.Unmarshal([]byte(res.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Games) != 1 || body.Games[0].Key != "pokemon" {
		t.Errorf("body = %s", res.Body)
	}
}

func TestCreateGame(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store, Token: "sekrit"}
	req := httpapi.Request{
		RouteKey: "POST /v1/games",
		Headers:  map[string]string{"authorization": "Bearer sekrit"},
		Body:     `{"key":"lorcana","label":"Disney Lorcana"}`,
	}
	res, err := api.Handler()(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 201 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	if len(store.added) != 1 || store.added[0] != "lorcana" {
		t.Errorf("added = %v", store.added)
	}
}

func TestCreateGameRequiresAuth(t *testing.T) {
	api := &API{Store: &fakeStore{}, Token: "sekrit"}
	res, _ := api.Handler()(context.Background(), httpapi.Request{
		RouteKey: "POST /v1/games",
		Body:     `{"key":"lorcana","label":"Disney Lorcana"}`,
	})
	if res.StatusCode != 401 {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
}

func TestCreateGameBadBodyAndConflict(t *testing.T) {
	api := &API{Store: &fakeStore{}}
	res, _ := api.Handler()(context.Background(), httpapi.Request{RouteKey: "POST /v1/games", Body: `{oops`})
	if res.StatusCode != 400 {
		t.Errorf("bad body status = %d, want 400", res.StatusCode)
	}
	api = &API{Store: &fakeStore{err: fmt.Errorf("%w: game", catalog.ErrConflict)}}
	res, _ = api.Handler()(context.Background(), httpapi.Request{RouteKey: "POST /v1/games", Body: `{"key":"x2","label":"X"}`})
	if res.StatusCode != 409 {
		t.Errorf("conflict status = %d, want 409", res.StatusCode)
	}
}

func TestHealthz(t *testing.T) {
	api := &API{Store: &fakeStore{}}
	res, _ := api.Handler()(context.Background(), httpapi.Request{RouteKey: "GET /healthz"})
	if res.StatusCode != 200 {
		t.Errorf("status = %d", res.StatusCode)
	}
}
