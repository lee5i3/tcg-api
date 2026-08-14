// The cards Lambda: search, read, and write a game's cards.
//
//	GET    /v1/games/{game}/sets/{set}/cards  — public, collector-number order
//	GET    /v1/games/{game}/cards             — public name search, ?query=
//	GET    /v1/games/{game}/cards/{card}      — public
//	POST   /v1/games/{game}/cards             — bearer token
//	PUT    /v1/games/{game}/cards/{card}      — bearer token
//	DELETE /v1/games/{game}/cards/{card}      — bearer token
//
// {game}/{set} accept GUID or key; {card} accepts GUID or TCGplayer id.
package main

import (
	"context"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

// Store is the slice of the catalog this function uses.
type Store interface {
	ResolveGame(ctx context.Context, idOrKey string) (catalog.Game, error)
	SearchCards(ctx context.Context, gameID, query string) ([]catalog.CardSummary, error)
	SetCards(ctx context.Context, gameID, setIDOrKey string) ([]catalog.CardSummary, error)
	GetCard(ctx context.Context, gameID, idOrKey string) (*catalog.CardDetail, error)
	CreateCard(ctx context.Context, game catalog.Game, in catalog.CardInput) (string, error)
	UpdateCard(ctx context.Context, game catalog.Game, idOrKey string, in catalog.CardInput) error
	DeleteCard(ctx context.Context, gameID, idOrKey string) error
}

type API struct {
	Store Store
	Token string
}

func (a *API) Handler() httpapi.Handler {
	return httpapi.Route(a.Token, map[string]httpapi.Handler{
		"GET /v1/games/{game}/sets/{set}/cards": a.setCards,
		"GET /v1/games/{game}/cards":            a.search,
		"GET /v1/games/{game}/cards/{card}":     a.get,
		"POST /v1/games/{game}/cards":           a.create,
		"PUT /v1/games/{game}/cards/{card}":     a.update,
		"DELETE /v1/games/{game}/cards/{card}":  a.remove,
	})
}

func (a *API) game(ctx context.Context, req httpapi.Request) (catalog.Game, error) {
	return a.Store.ResolveGame(ctx, req.PathParameters["game"])
}

func (a *API) setCards(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	cards, err := a.Store.SetCards(ctx, game.ID, req.PathParameters["set"])
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(200, map[string]any{"cards": cards})
}

func (a *API) search(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	cards, err := a.Store.SearchCards(ctx, game.ID, req.QueryStringParameters["query"])
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(200, map[string]any{"cards": cards})
}

func (a *API) get(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	card, err := a.Store.GetCard(ctx, game.ID, req.PathParameters["card"])
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(200, map[string]any{"card": card})
}

func (a *API) create(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	var in catalog.CardInput
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	id, err := a.Store.CreateCard(ctx, game, in)
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(201, map[string]string{"id": id})
}

func (a *API) update(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	var in catalog.CardInput
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	if err := a.Store.UpdateCard(ctx, game, req.PathParameters["card"], in); err != nil {
		return httpapi.Error(err)
	}
	return httpapi.Response{StatusCode: 204}, nil
}

func (a *API) remove(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	if err := a.Store.DeleteCard(ctx, game.ID, req.PathParameters["card"]); err != nil {
		return httpapi.Error(err)
	}
	return httpapi.Response{StatusCode: 204}, nil
}
