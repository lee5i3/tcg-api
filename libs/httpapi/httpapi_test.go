package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
)

func TestErrorStatusMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{fmt.Errorf("%w: game", catalog.ErrNotFound), 404},
		{fmt.Errorf("%w: key", catalog.ErrInvalid), 400},
		{fmt.Errorf("%w: set", catalog.ErrConflict), 409},
		{errors.New("dynamodb exploded"), 500},
	} {
		res, err := Error(tc.err)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != tc.status {
			t.Errorf("Error(%v) status = %d, want %d", tc.err, res.StatusCode, tc.status)
		}
		var body map[string]string
		if err := json.Unmarshal([]byte(res.Body), &body); err != nil || body["error"] == "" {
			t.Errorf("Error(%v) body = %q", tc.err, res.Body)
		}
		// Internal details must not leak.
		if tc.status == 500 && body["error"] != "internal error" {
			t.Errorf("500 leaked internals: %q", body["error"])
		}
	}
}

func TestAuthorized(t *testing.T) {
	req := func(header string) Request {
		return Request{Headers: map[string]string{"authorization": header}}
	}
	if !Authorized(req(""), "") {
		t.Error("empty configured token must disable auth")
	}
	if !Authorized(req("Bearer sekrit"), "sekrit") {
		t.Error("valid token rejected")
	}
	for _, bad := range []string{"", "Bearer wrong", "sekrit", "bearer sekrit"} {
		if Authorized(req(bad), "sekrit") {
			t.Errorf("header %q accepted", bad)
		}
	}
	// Capitalized header name (API Gateway lowercases, but be tolerant).
	if !Authorized(Request{Headers: map[string]string{"Authorization": "Bearer sekrit"}}, "sekrit") {
		t.Error("capitalized Authorization header rejected")
	}
}

func TestRouteAuthAndDispatch(t *testing.T) {
	ok := func(ctx context.Context, req Request) (Response, error) {
		return JSON(200, map[string]string{"hit": req.RouteKey})
	}
	h := Route("sekrit", map[string]Handler{
		"GET /v1/games":  ok,
		"POST /v1/games": ok,
	})

	// Public read needs no token.
	res, _ := h(context.Background(), Request{RouteKey: "GET /v1/games"})
	if res.StatusCode != 200 {
		t.Errorf("GET status = %d", res.StatusCode)
	}
	// Write without token is 401.
	res, _ = h(context.Background(), Request{RouteKey: "POST /v1/games"})
	if res.StatusCode != 401 {
		t.Errorf("unauthenticated POST status = %d, want 401", res.StatusCode)
	}
	// Write with token passes.
	res, _ = h(context.Background(), Request{
		RouteKey: "POST /v1/games",
		Headers:  map[string]string{"authorization": "Bearer sekrit"},
	})
	if res.StatusCode != 200 {
		t.Errorf("authenticated POST status = %d", res.StatusCode)
	}
	// Unknown route is 404.
	res, _ = h(context.Background(), Request{RouteKey: "GET /nope"})
	if res.StatusCode != 404 {
		t.Errorf("unknown route status = %d", res.StatusCode)
	}
}
