// The auth Lambda: end-user accounts for the app, plus the admin token check.
//
//	POST /v1/auth/register — public; create an account → 201 {token, user}
//	POST /v1/auth/login    — public; email/password → 200 {token, user}
//	GET  /v1/auth/me       — user JWT in Authorization → 200 {user}
//	POST /v1/auth/check    — ADMIN bearer token (API_TOKEN); 204 when valid.
//	                         The admin site's login uses this.
//
// User sessions are JWTs (HS256, USER_JWT_SECRET) minted here and verified
// here — no session storage. Admin auth stays the shared API_TOKEN model.
package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/lee5i3/tcg-api/libs/httpapi"
	useraccounts "github.com/lee5i3/tcg-api/libs/user-accounts"
)

// Store is the slice of the account store this function uses.
type Store interface {
	Register(ctx context.Context, email, password, name string) (useraccounts.User, error)
	Authenticate(ctx context.Context, email, password string) (useraccounts.User, error)
	GetUser(ctx context.Context, id string) (useraccounts.User, error)
	FindOrCreateOAuthUser(ctx context.Context, provider, subject, email, name string) (useraccounts.User, error)
}

// Tokens mints and verifies user session tokens and OAuth state.
type Tokens interface {
	Mint(user useraccounts.User) (string, error)
	Verify(raw string) (string, error)
	MintState(provider, redirect string) (string, error)
	VerifyState(raw string) (provider, redirect string, err error)
}

type API struct {
	Store  Store
	Tokens Tokens
	Token  string // admin API token, for /v1/auth/check
	OAuth  *OAuth
	Log    *slog.Logger
}

func (a *API) Handler() httpapi.Handler {
	return httpapi.Dispatch(a.Token,
		map[string]httpapi.Handler{
			"POST /v1/auth/check":                    a.adminCheck,
			"GET /v1/auth/me":                        a.me,
			"GET /v1/auth/providers":                 a.providers,
			"GET /v1/auth/oauth/{provider}/start":    a.oauthStart,
			"GET /v1/auth/oauth/{provider}/callback": a.oauthCallback,
		},
		map[string]httpapi.Handler{
			"POST /v1/auth/register": a.register,
			"POST /v1/auth/login":    a.login,
			// Apple sends its callback as a form POST (response_mode=form_post).
			"POST /v1/auth/oauth/{provider}/callback": a.oauthCallback,
		})
}

// adminCheck exists so the admin site can validate the API token up front;
// Dispatch has already enforced the bearer token by the time this runs.
func (a *API) adminCheck(ctx context.Context, _ httpapi.Request) (httpapi.Response, error) {
	return httpapi.Response{StatusCode: 204}, nil
}

// authError maps account errors onto HTTP statuses without leaking which
// half of a credential pair was wrong.
func authError(err error) (httpapi.Response, error) {
	switch {
	case errors.Is(err, useraccounts.ErrBadCredentials):
		return httpapi.JSON(401, map[string]string{"error": useraccounts.ErrBadCredentials.Error()})
	case errors.Is(err, useraccounts.ErrEmailTaken):
		return httpapi.JSON(409, map[string]string{"error": useraccounts.ErrEmailTaken.Error()})
	case errors.Is(err, useraccounts.ErrInvalid):
		return httpapi.JSON(400, map[string]string{"error": err.Error()})
	default:
		return httpapi.Error(err) // opaque 500
	}
}

func (a *API) session(status int, user useraccounts.User) (httpapi.Response, error) {
	token, err := a.Tokens.Mint(user)
	if err != nil {
		return httpapi.Error(err)
	}
	return httpapi.JSON(status, map[string]any{"token": token, "user": user})
}

func (a *API) register(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	user, err := a.Store.Register(ctx, in.Email, in.Password, in.Name)
	if err != nil {
		return authError(err)
	}
	return a.session(201, user)
}

func (a *API) login(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpapi.Body(req, &in); err != nil {
		return httpapi.ErrBadRequest("invalid JSON body")
	}
	user, err := a.Store.Authenticate(ctx, in.Email, in.Password)
	if err != nil {
		return authError(err)
	}
	return a.session(200, user)
}

// me validates the caller's session token and returns their account.
func (a *API) me(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	header := req.Headers["authorization"]
	if header == "" {
		header = req.Headers["Authorization"]
	}
	raw, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return httpapi.JSON(401, map[string]string{"error": "missing bearer token"})
	}
	userID, err := a.Tokens.Verify(raw)
	if err != nil {
		return httpapi.JSON(401, map[string]string{"error": ErrBadToken.Error()})
	}
	user, err := a.Store.GetUser(ctx, userID)
	if err != nil {
		// The account may have been deleted since the token was minted.
		return httpapi.JSON(401, map[string]string{"error": ErrBadToken.Error()})
	}
	return httpapi.JSON(200, map[string]any{"user": user})
}
