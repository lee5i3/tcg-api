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
	searchQuery string
	setRef      string
	created     []catalog.CardInput
	updated     map[string]catalog.CardInput
	deleted     []string
}

func (f *fakeStore) ResolveGame(_ context.Context, idOrKey string) (catalog.Game, error) {
	if idOrKey != "pokemon" {
		return catalog.Game{}, fmt.Errorf("%w: game %q", catalog.ErrNotFound, idOrKey)
	}
	return catalog.Game{ID: "game-1", Key: "pokemon"}, nil
}

func (f *fakeStore) SearchCards(_ context.Context, _, query string) ([]catalog.CardSummary, error) {
	f.searchQuery = query
	return []catalog.CardSummary{{ID: "card-1", Name: "Zapdos"}}, nil
}

func (f *fakeStore) SetCards(_ context.Context, _, setIDOrKey string) ([]catalog.CardSummary, error) {
	f.setRef = setIDOrKey
	return []catalog.CardSummary{{ID: "card-1", Name: "Zapdos", Number: "145"}}, nil
}

func (f *fakeStore) GetCard(_ context.Context, _, idOrKey string) (*catalog.CardDetail, error) {
	if idOrKey == "missing" {
		return nil, fmt.Errorf("%w: card %q", catalog.ErrNotFound, idOrKey)
	}
	return &catalog.CardDetail{ID: idOrKey, Name: "Zapdos"}, nil
}

func (f *fakeStore) CreateCard(_ context.Context, _ catalog.Game, in catalog.CardInput) (string, error) {
	if in.SetID == "missing" {
		return "", fmt.Errorf("%w: set %q does not exist", catalog.ErrInvalid, in.SetID)
	}
	f.created = append(f.created, in)
	return "card-new", nil
}

func (f *fakeStore) UpdateCard(_ context.Context, _ catalog.Game, idOrKey string, in catalog.CardInput) error {
	if f.updated == nil {
		f.updated = map[string]catalog.CardInput{}
	}
	f.updated[idOrKey] = in
	return nil
}

func (f *fakeStore) DeleteCard(_ context.Context, _, idOrKey string) error {
	f.deleted = append(f.deleted, idOrKey)
	return nil
}

func request(routeKey string, path, query map[string]string, body string) httpapi.Request {
	return httpapi.Request{
		RouteKey:              routeKey,
		PathParameters:        path,
		QueryStringParameters: query,
		Body:                  body,
	}
}

func TestSetCards(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store}
	res, err := api.Handler()(context.Background(), request(
		"GET /v1/games/{game}/sets/{set}/cards",
		map[string]string{"game": "pokemon", "set": "sv3pt5"}, nil, ""))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || store.setRef != "sv3pt5" {
		t.Errorf("status = %d, setRef = %q", res.StatusCode, store.setRef)
	}
}

func TestSearch(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store}
	res, _ := api.Handler()(context.Background(), request(
		"GET /v1/games/{game}/cards",
		map[string]string{"game": "pokemon"},
		map[string]string{"query": "zapdos"}, ""))
	if res.StatusCode != 200 || store.searchQuery != "zapdos" {
		t.Errorf("status = %d, query = %q", res.StatusCode, store.searchQuery)
	}
	var body struct {
		Cards []catalog.CardSummary `json:"cards"`
	}
	if err := json.Unmarshal([]byte(res.Body), &body); err != nil || len(body.Cards) != 1 {
		t.Errorf("body = %s", res.Body)
	}
}

func TestGetCard(t *testing.T) {
	api := &API{Store: &fakeStore{}}
	res, _ := api.Handler()(context.Background(), request(
		"GET /v1/games/{game}/cards/{card}",
		map[string]string{"game": "pokemon", "card": "card-1"}, nil, ""))
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	res, _ = api.Handler()(context.Background(), request(
		"GET /v1/games/{game}/cards/{card}",
		map[string]string{"game": "pokemon", "card": "missing"}, nil, ""))
	if res.StatusCode != 404 {
		t.Errorf("missing card status = %d, want 404", res.StatusCode)
	}
}

func TestCreateCard(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store, Token: "sekrit"}
	authed := map[string]string{"authorization": "Bearer sekrit"}

	req := request("POST /v1/games/{game}/cards",
		map[string]string{"game": "pokemon"}, nil,
		`{"setId":"sv3pt5","name":"Zapdos","number":"145"}`)
	req.Headers = authed
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 201 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	if len(store.created) != 1 || store.created[0].Name != "Zapdos" {
		t.Errorf("created = %+v", store.created)
	}

	// Unknown set surfaces as 400.
	req.Body = `{"setId":"missing","name":"Lost"}`
	res, _ = api.Handler()(context.Background(), req)
	if res.StatusCode != 400 {
		t.Errorf("bad set status = %d, want 400", res.StatusCode)
	}

	// No token → 401.
	req.Headers = nil
	res, _ = api.Handler()(context.Background(), req)
	if res.StatusCode != 401 {
		t.Errorf("unauthenticated status = %d, want 401", res.StatusCode)
	}
}

func TestUpdateAndDeleteCard(t *testing.T) {
	store := &fakeStore{}
	api := &API{Store: store}

	res, _ := api.Handler()(context.Background(), request(
		"PUT /v1/games/{game}/cards/{card}",
		map[string]string{"game": "pokemon", "card": "card-1"}, nil,
		`{"setId":"sv3pt5","name":"Zapdos ex"}`))
	if res.StatusCode != 204 || store.updated["card-1"].Name != "Zapdos ex" {
		t.Errorf("update status = %d, updated = %+v", res.StatusCode, store.updated)
	}

	res, _ = api.Handler()(context.Background(), request(
		"DELETE /v1/games/{game}/cards/{card}",
		map[string]string{"game": "pokemon", "card": "501773"}, nil, ""))
	if res.StatusCode != 204 || len(store.deleted) != 1 || store.deleted[0] != "501773" {
		t.Errorf("delete status = %d, deleted = %v", res.StatusCode, store.deleted)
	}
}
