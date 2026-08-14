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
	sets      []catalog.SetSummary
	lastQuery string
	updated   map[string]catalog.SetInput
	deleted   []string
}

func (f *fakeStore) ResolveGame(_ context.Context, idOrKey string) (catalog.Game, error) {
	if idOrKey != "pokemon" && idOrKey != "game-1" {
		return catalog.Game{}, fmt.Errorf("%w: game %q", catalog.ErrNotFound, idOrKey)
	}
	return catalog.Game{ID: "game-1", Key: "pokemon"}, nil
}

func (f *fakeStore) ListSets(_ context.Context, gameID, query string) ([]catalog.SetSummary, error) {
	f.lastQuery = query
	return f.sets, nil
}

func (f *fakeStore) CreateSet(_ context.Context, gameID string, in catalog.SetInput) (string, error) {
	if in.Key == "taken" {
		return "", fmt.Errorf("%w: set %q", catalog.ErrConflict, in.Key)
	}
	return "set-1", nil
}

func (f *fakeStore) UpdateSet(_ context.Context, gameID, idOrKey string, in catalog.SetInput) error {
	if f.updated == nil {
		f.updated = map[string]catalog.SetInput{}
	}
	f.updated[idOrKey] = in
	return nil
}

func (f *fakeStore) DeleteSet(_ context.Context, gameID, idOrKey string) (int, error) {
	f.deleted = append(f.deleted, idOrKey)
	return 7, nil
}

func request(routeKey string, path map[string]string, query map[string]string, body string) httpapi.Request {
	return httpapi.Request{
		RouteKey:              routeKey,
		PathParameters:        path,
		QueryStringParameters: query,
		Body:                  body,
	}
}

func TestListSetsPassesQuery(t *testing.T) {
	store := &fakeStore{sets: []catalog.SetSummary{{ID: "set-1", Key: "sv3pt5", Name: "151"}}}
	api := &API{Store: store}
	res, err := api.Handler()(context.Background(), request(
		"GET /v1/games/{game}/sets",
		map[string]string{"game": "pokemon"},
		map[string]string{"query": "151"}, ""))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || store.lastQuery != "151" {
		t.Errorf("status = %d, query = %q", res.StatusCode, store.lastQuery)
	}
	var body struct {
		Sets []catalog.SetSummary `json:"sets"`
	}
	if err := json.Unmarshal([]byte(res.Body), &body); err != nil || len(body.Sets) != 1 {
		t.Errorf("body = %s", res.Body)
	}
}

func TestListSetsUnknownGame(t *testing.T) {
	api := &API{Store: &fakeStore{}}
	res, _ := api.Handler()(context.Background(), request(
		"GET /v1/games/{game}/sets", map[string]string{"game": "nope"}, nil, ""))
	if res.StatusCode != 404 {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}

func TestCreateSet(t *testing.T) {
	api := &API{Store: &fakeStore{}}
	res, _ := api.Handler()(context.Background(), request(
		"POST /v1/games/{game}/sets", map[string]string{"game": "pokemon"}, nil,
		`{"key":"sv3pt5","name":"151"}`))
	if res.StatusCode != 201 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	var body map[string]string
	_ = json.Unmarshal([]byte(res.Body), &body)
	if body["id"] != "set-1" {
		t.Errorf("body = %s", res.Body)
	}

	// Conflict propagates as 409.
	res, _ = api.Handler()(context.Background(), request(
		"POST /v1/games/{game}/sets", map[string]string{"game": "pokemon"}, nil,
		`{"key":"taken","name":"X"}`))
	if res.StatusCode != 409 {
		t.Errorf("conflict status = %d", res.StatusCode)
	}
}

func TestUpdateSet(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store}
	res, _ := api.Handler()(context.Background(), request(
		"PUT /v1/games/{game}/sets/{set}",
		map[string]string{"game": "pokemon", "set": "sv3pt5"}, nil,
		`{"name":"151 (renamed)"}`))
	if res.StatusCode != 204 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	if store.updated["sv3pt5"].Name != "151 (renamed)" {
		t.Errorf("updated = %+v", store.updated)
	}
}

func TestDeleteSet(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store}
	res, _ := api.Handler()(context.Background(), request(
		"DELETE /v1/games/{game}/sets/{set}",
		map[string]string{"game": "pokemon", "set": "sv3pt5"}, nil, ""))
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var body map[string]int
	_ = json.Unmarshal([]byte(res.Body), &body)
	if body["cardsDeleted"] != 7 || len(store.deleted) != 1 {
		t.Errorf("body = %s, deleted = %v", res.Body, store.deleted)
	}
}
