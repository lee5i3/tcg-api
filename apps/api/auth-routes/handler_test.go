package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/lee5i3/tcg-api/libs/httpapi"
	useraccounts "github.com/lee5i3/tcg-api/libs/user-accounts"
)

type fakeStore struct {
	users     map[string]useraccounts.User // id → user
	oauthSeen string                       // provider/subject of the last OAuth resolve
}

func (f *fakeStore) Register(_ context.Context, email, password, name string) (useraccounts.User, error) {
	if email == "taken@example.com" {
		return useraccounts.User{}, useraccounts.ErrEmailTaken
	}
	if len(password) < 8 {
		return useraccounts.User{}, useraccounts.ErrInvalid
	}
	u := useraccounts.User{ID: "user-1", Email: email, Name: name}
	f.users = map[string]useraccounts.User{u.ID: u}
	return u, nil
}

func (f *fakeStore) Authenticate(_ context.Context, email, password string) (useraccounts.User, error) {
	if email == "lee@example.com" && password == "hunter2secure" {
		return useraccounts.User{ID: "user-1", Email: email}, nil
	}
	return useraccounts.User{}, useraccounts.ErrBadCredentials
}

func (f *fakeStore) GetUser(_ context.Context, id string) (useraccounts.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return useraccounts.User{}, useraccounts.ErrNotFound
}

func testAPI() *API {
	return &API{
		Store:  &fakeStore{users: map[string]useraccounts.User{"user-1": {ID: "user-1", Email: "lee@example.com"}}},
		Tokens: NewJWTTokens("test-secret"),
		Token:  "admin-token",
		OAuth:  &OAuth{Providers: map[string]*OAuthProvider{}},
		Log:    slog.New(slog.DiscardHandler),
	}
}

func request(routeKey, body string, headers map[string]string) httpapi.Request {
	return httpapi.Request{RouteKey: routeKey, Body: body, Headers: headers}
}

func TestRegister(t *testing.T) {
	api := testAPI()
	res, err := api.Handler()(context.Background(), request(
		"POST /v1/auth/register", `{"email":"new@example.com","password":"hunter2secure","name":"Lee"}`, nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 201 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	var body struct {
		Token string            `json:"token"`
		User  useraccounts.User `json:"user"`
	}
	if err := json.Unmarshal([]byte(res.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Token == "" || body.User.Email != "new@example.com" {
		t.Errorf("body = %s", res.Body)
	}
	// The minted token verifies back to the user.
	if id, err := api.Tokens.Verify(body.Token); err != nil || id != "user-1" {
		t.Errorf("verify minted token: %q, %v", id, err)
	}
}

func TestRegisterErrors(t *testing.T) {
	api := testAPI()
	res, _ := api.Handler()(context.Background(), request(
		"POST /v1/auth/register", `{"email":"taken@example.com","password":"hunter2secure"}`, nil))
	if res.StatusCode != 409 {
		t.Errorf("taken email status = %d", res.StatusCode)
	}
	res, _ = api.Handler()(context.Background(), request(
		"POST /v1/auth/register", `{"email":"x@example.com","password":"short"}`, nil))
	if res.StatusCode != 400 {
		t.Errorf("short password status = %d", res.StatusCode)
	}
	res, _ = api.Handler()(context.Background(), request("POST /v1/auth/register", `{oops`, nil))
	if res.StatusCode != 400 {
		t.Errorf("bad body status = %d", res.StatusCode)
	}
}

func TestRegisterAndLoginArePublic(t *testing.T) {
	// No admin bearer token anywhere — public routes must still work.
	api := testAPI()
	res, _ := api.Handler()(context.Background(), request(
		"POST /v1/auth/login", `{"email":"lee@example.com","password":"hunter2secure"}`, nil))
	if res.StatusCode != 200 {
		t.Errorf("login without admin token status = %d, want 200", res.StatusCode)
	}
}

func TestLoginWrongCredentials(t *testing.T) {
	api := testAPI()
	res, _ := api.Handler()(context.Background(), request(
		"POST /v1/auth/login", `{"email":"lee@example.com","password":"wrong"}`, nil))
	if res.StatusCode != 401 {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
	var body map[string]string
	_ = json.Unmarshal([]byte(res.Body), &body)
	if body["error"] != "invalid email or password" {
		t.Errorf("error = %q (must not leak which half was wrong)", body["error"])
	}
}

func TestMe(t *testing.T) {
	api := testAPI()
	token, _ := api.Tokens.Mint(useraccounts.User{ID: "user-1"})

	res, _ := api.Handler()(context.Background(), request(
		"GET /v1/auth/me", "", map[string]string{"authorization": "Bearer " + token}))
	if res.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	// Missing / garbage / deleted-user tokens are all 401.
	res, _ = api.Handler()(context.Background(), request("GET /v1/auth/me", "", nil))
	if res.StatusCode != 401 {
		t.Errorf("missing token status = %d", res.StatusCode)
	}
	res, _ = api.Handler()(context.Background(), request(
		"GET /v1/auth/me", "", map[string]string{"authorization": "Bearer not-a-jwt"}))
	if res.StatusCode != 401 {
		t.Errorf("garbage token status = %d", res.StatusCode)
	}
	ghost, _ := api.Tokens.Mint(useraccounts.User{ID: "deleted-user"})
	res, _ = api.Handler()(context.Background(), request(
		"GET /v1/auth/me", "", map[string]string{"authorization": "Bearer " + ghost}))
	if res.StatusCode != 401 {
		t.Errorf("deleted user status = %d", res.StatusCode)
	}
}

func TestAdminCheck(t *testing.T) {
	api := testAPI()
	res, _ := api.Handler()(context.Background(), request("POST /v1/auth/check", "", nil))
	if res.StatusCode != 401 {
		t.Errorf("no token status = %d", res.StatusCode)
	}
	res, _ = api.Handler()(context.Background(), request(
		"POST /v1/auth/check", "", map[string]string{"authorization": "Bearer admin-token"}))
	if res.StatusCode != 204 {
		t.Errorf("valid token status = %d", res.StatusCode)
	}
}

func TestTokenExpiry(t *testing.T) {
	tokens := NewJWTTokens("test-secret")
	tokens.now = func() time.Time { return time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC) }
	token, err := tokens.Mint(useraccounts.User{ID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if id, err := tokens.Verify(token); err != nil || id != "user-1" {
		t.Errorf("fresh token: %q, %v", id, err)
	}
	// 8 days later the 7-day token is expired.
	tokens.now = func() time.Time { return time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC) }
	if _, err := tokens.Verify(token); err == nil {
		t.Error("expired token verified")
	}
	// A token signed with a different secret is rejected.
	other, _ := NewJWTTokens("other-secret").Mint(useraccounts.User{ID: "user-1"})
	tokens.now = func() time.Time { return time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC) }
	if _, err := tokens.Verify(other); err == nil {
		t.Error("token with wrong secret verified")
	}
}
