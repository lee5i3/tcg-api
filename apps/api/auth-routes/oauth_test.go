package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lee5i3/tcg-api/libs/httpapi"
	useraccounts "github.com/lee5i3/tcg-api/libs/user-accounts"
)

func (f *fakeStore) FindOrCreateOAuthUser(_ context.Context, provider, subject, email, name string) (useraccounts.User, error) {
	if email == "no-account@example.com" {
		return useraccounts.User{}, useraccounts.ErrInvalid
	}
	u := useraccounts.User{ID: "user-oauth-1", Email: email, Name: name}
	if f.users == nil {
		f.users = map[string]useraccounts.User{}
	}
	f.users[u.ID] = u
	f.oauthSeen = provider + "/" + subject
	return u, nil
}

// fakeIDToken builds an unsigned-but-shaped JWT with the given claims.
func fakeIDToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	return head + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// oauthTestAPI wires a google-like provider whose endpoints are httptest
// servers.
func oauthTestAPI(t *testing.T, tokenHandler http.HandlerFunc) (*API, *fakeStore) {
	t.Helper()
	tokenSrv := httptest.NewServer(tokenHandler)
	t.Cleanup(tokenSrv.Close)
	store := &fakeStore{users: map[string]useraccounts.User{}}
	api := &API{
		Store:  store,
		Tokens: NewJWTTokens("test-secret"),
		Token:  "admin-token",
		Log:    slog.New(slog.DiscardHandler),
		OAuth: &OAuth{
			AppURL: "https://app.example.com",
			Providers: map[string]*OAuthProvider{
				"google": {
					ID: "google", Label: "Google", ClientID: "cid", ClientSecret: "csecret",
					AuthURL:  "https://provider.example.com/auth",
					TokenURL: tokenSrv.URL,
					Scopes:   "openid email profile",
				},
			},
		},
	}
	return api, store
}

func TestProvidersListsConfigured(t *testing.T) {
	api, _ := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {})
	res, _ := api.Handler()(context.Background(), request("GET /v1/auth/providers", "", nil))
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var body struct {
		Providers []struct{ ID, Label string }
	}
	if err := json.Unmarshal([]byte(res.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Providers) != 1 || body.Providers[0].ID != "google" {
		t.Errorf("providers = %+v", body.Providers)
	}
}

func TestOAuthStartRedirects(t *testing.T) {
	api, _ := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {})
	req := httpapi.Request{
		RouteKey:              "GET /v1/auth/oauth/{provider}/start",
		PathParameters:        map[string]string{"provider": "google"},
		QueryStringParameters: map[string]string{"redirect": "/g/pokemon"},
	}
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 302 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	loc, err := url.Parse(res.Headers["Location"])
	if err != nil {
		t.Fatal(err)
	}
	q := loc.Query()
	if q.Get("client_id") != "cid" || q.Get("response_type") != "code" {
		t.Errorf("authorize query = %v", q)
	}
	if q.Get("redirect_uri") != "https://app.example.com/v1/auth/oauth/google/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	// The state must round-trip with the provider and return path.
	provider, redirect, err := api.Tokens.VerifyState(q.Get("state"))
	if err != nil || provider != "google" || redirect != "/g/pokemon" {
		t.Errorf("state = %q %q %v", provider, redirect, err)
	}
	// Unknown provider is a 404.
	req.PathParameters["provider"] = "myspace"
	res, _ = api.Handler()(context.Background(), req)
	if res.StatusCode != 404 {
		t.Errorf("unknown provider status = %d", res.StatusCode)
	}
}

func TestOAuthCallbackHappyPath(t *testing.T) {
	idToken := ""
	api, store := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("code") != "auth-code" || r.PostForm.Get("client_secret") != "csecret" {
			http.Error(w, "bad exchange", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "at",
			"id_token":     idToken,
		})
	})
	idToken = fakeIDToken(t, map[string]any{"sub": "goog-42", "email": "lee@example.com", "name": "Lee"})

	state, _ := api.Tokens.MintState("google", "/g/pokemon")
	req := httpapi.Request{
		RouteKey:              "GET /v1/auth/oauth/{provider}/callback",
		PathParameters:        map[string]string{"provider": "google"},
		QueryStringParameters: map[string]string{"code": "auth-code", "state": state},
	}
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 302 {
		t.Fatalf("status = %d, body = %s", res.StatusCode, res.Body)
	}
	loc := res.Headers["Location"]
	if !strings.HasPrefix(loc, "https://app.example.com/auth/callback#token=") {
		t.Fatalf("location = %q", loc)
	}
	if !strings.Contains(loc, "redirect=%2Fg%2Fpokemon") {
		t.Errorf("return path missing: %q", loc)
	}
	if store.oauthSeen != "google/goog-42" {
		t.Errorf("identity = %q", store.oauthSeen)
	}
	// The token in the fragment is a valid session for the user.
	frag := strings.SplitN(loc, "#", 2)[1]
	vals, _ := url.ParseQuery(frag)
	if id, err := api.Tokens.Verify(vals.Get("token")); err != nil || id != "user-oauth-1" {
		t.Errorf("session from fragment: %q %v", id, err)
	}
}

func TestOAuthCallbackRejectsBadState(t *testing.T) {
	api, _ := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {})
	// State minted for another provider must not pass.
	state, _ := api.Tokens.MintState("facebook", "/")
	req := httpapi.Request{
		RouteKey:              "GET /v1/auth/oauth/{provider}/callback",
		PathParameters:        map[string]string{"provider": "google"},
		QueryStringParameters: map[string]string{"code": "auth-code", "state": state},
	}
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 302 || !strings.Contains(res.Headers["Location"], "/login?error=") {
		t.Errorf("bad state: %d %q", res.StatusCode, res.Headers["Location"])
	}
	// A session token is not a state token.
	session, _ := api.Tokens.Mint(useraccounts.User{ID: "user-1"})
	if _, _, err := api.Tokens.VerifyState(session); err == nil {
		t.Error("session token accepted as state")
	}
	// And state is never a session.
	if _, err := api.Tokens.Verify(state); err == nil {
		t.Error("state token accepted as session")
	}
}

func TestOAuthCallbackProviderDenied(t *testing.T) {
	api, _ := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {})
	req := httpapi.Request{
		RouteKey:              "GET /v1/auth/oauth/{provider}/callback",
		PathParameters:        map[string]string{"provider": "google"},
		QueryStringParameters: map[string]string{"error": "access_denied"},
	}
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 302 || !strings.Contains(res.Headers["Location"], "cancelled") {
		t.Errorf("denied: %d %q", res.StatusCode, res.Headers["Location"])
	}
}

func TestOAuthCallbackFormPost(t *testing.T) {
	// Apple-style: params arrive form-encoded in a POST body.
	idToken := ""
	api, _ := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": idToken})
	})
	idToken = fakeIDToken(t, map[string]any{"sub": "appl-7", "email": "lee@example.com"})

	state, _ := api.Tokens.MintState("google", "/")
	body := url.Values{"code": {"auth-code"}, "state": {state}}.Encode()
	req := httpapi.Request{
		RouteKey:       "POST /v1/auth/oauth/{provider}/callback",
		PathParameters: map[string]string{"provider": "google"},
		Body:           body,
	}
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 302 || !strings.Contains(res.Headers["Location"], "/auth/callback#token=") {
		t.Errorf("form_post callback: %d %q", res.StatusCode, res.Headers["Location"])
	}
}

func TestOAuthCallbackMissingEmail(t *testing.T) {
	idToken := ""
	api, _ := oauthTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": idToken})
	})
	idToken = fakeIDToken(t, map[string]any{"sub": "goog-1"})

	state, _ := api.Tokens.MintState("google", "/")
	req := httpapi.Request{
		RouteKey:              "GET /v1/auth/oauth/{provider}/callback",
		PathParameters:        map[string]string{"provider": "google"},
		QueryStringParameters: map[string]string{"code": "auth-code", "state": state},
	}
	res, _ := api.Handler()(context.Background(), req)
	if res.StatusCode != 302 || !strings.Contains(res.Headers["Location"], "email") {
		t.Errorf("missing email: %d %q", res.StatusCode, res.Headers["Location"])
	}
}

func TestLoadProvidersFromEnv(t *testing.T) {
	env := map[string]string{
		"GOOGLE_CLIENT_ID": "g", "GOOGLE_CLIENT_SECRET": "gs",
		"APPLE_CLIENT_ID": "a", // secret missing → apple disabled
	}
	providers := LoadProvidersFromEnv(func(k string) string { return env[k] })
	if len(providers) != 1 {
		t.Fatalf("providers = %v", providers)
	}
	if p := providers["google"]; p == nil || !strings.Contains(p.AuthURL, "accounts.google.com") {
		t.Errorf("google = %+v", p)
	}
}

func TestSanitizeReturnPath(t *testing.T) {
	for in, want := range map[string]string{
		"":                "/",
		"/g/pokemon":      "/g/pokemon",
		"//evil.com":      "/",
		"https://evil.io": "/",
	} {
		if got := sanitizeReturnPath(in); got != want {
			t.Errorf("sanitizeReturnPath(%q) = %q, want %q", in, got, want)
		}
	}
}
