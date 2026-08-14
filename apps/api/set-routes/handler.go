// The sets Lambda: list, create, update, and delete a game's sets.
//
//	GET    /v1/games/{game}/sets        — public, ?query= filters by name
//	POST   /v1/games/{game}/sets        — bearer token
//	PUT    /v1/games/{game}/sets/{set}  — bearer token
//	DELETE /v1/games/{game}/sets/{set}  — bearer token
//
// {game} and {set} accept the GUID or the immutable catalog key.
package main

import (
	"context"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

// Store is the slice of the catalog this function uses.
type Store interface {
	ResolveGame(ctx context.Context, idOrKey string) (catalog.Game, error)
	ListSets(ctx context.Context, gameID, query string) ([]catalog.SetSummary, error)
	CreateSet(ctx context.Context, gameID string, in catalog.SetInput) (string, error)
	UpdateSet(ctx context.Context, gameID, idOrKey string, in catalog.SetInput) error
	DeleteSet(ctx context.Context, gameID, idOrKey string) (int, error)
}

type API struct {
	Store Store
	Token string
}

func (a *API) Handler() httpapi.Handler {
	return httpapi.Route(a.Token, map[string]httpapi.Handler{
		"GET /v1/games/{game}/sets":          a.list,
		"POST /v1/games/{game}/sets":         a.create,
		"PUT /v1/games/{game}/sets/{set}":    a.update,
		"DELETE /v1/games/{game}/sets/{set}": a.remove,
	})
}

func (a *API) game(ctx context.Context, req httpapi.Request) (catalog.Game, error) {
	return a.Store.ResolveGame(ctx, req.PathParameters["game"])
}

func (a *API) list(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	sets, err := a.Store.ListSets(ctx, game.ID, req.QueryStringParameters["query"])
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(200, map[string]any{"sets": sets})
}

func (a *API) create(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	var in catalog.SetInput
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	id, err := a.Store.CreateSet(ctx, game.ID, in)
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
	var in catalog.SetInput
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	if err := a.Store.UpdateSet(ctx, game.ID, req.PathParameters["set"], in); err != nil {
		return httpapi.Error(err)
	}
	return httpapi.Response{StatusCode: 204}, nil
}

func (a *API) remove(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	game, err := a.game(ctx, req)
	if err != nil {
		return httpapi.Error(err)
	}
	n, err := a.Store.DeleteSet(ctx, game.ID, req.PathParameters["set"])
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(200, map[string]int{"cardsDeleted": n})
}
