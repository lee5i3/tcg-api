// The games Lambda: list and create games, plus the liveness probe.
//
//	GET  /healthz   — liveness probe
//	GET  /v1/games  — public
//	POST /v1/games  — bearer token
package main

import (
	"context"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

// Store is the slice of the catalog this function uses.
type Store interface {
	Games(ctx context.Context) ([]catalog.Game, error)
	AddGame(ctx context.Context, key, label string, language *string) (catalog.Game, error)
}

type API struct {
	Store Store
	Token string
}

func (a *API) Handler() httpapi.Handler {
	return httpapi.Route(a.Token, map[string]httpapi.Handler{
		"GET /healthz":   a.health,
		"GET /v1/games":  a.list,
		"POST /v1/games": a.create,
	})
}

func (a *API) health(ctx context.Context, _ httpapi.Request) (httpapi.Response, error) {
	return httpapi.JSON(200, map[string]string{"status": "ok"})
}

func (a *API) list(ctx context.Context, _ httpapi.Request) (httpapi.Response, error) {
	games, err := a.Store.Games(ctx)
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(200, map[string]any{"games": games})
}

func (a *API) create(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	var in struct {
		Key      string  `json:"key"`
		Label    string  `json:"label"`
		Language *string `json:"language"`
	}
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	game, err := a.Store.AddGame(ctx, in.Key, in.Label, in.Language)
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(201, map[string]any{"game": game})
}
