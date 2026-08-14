package main

// Social sign-in: the standard OAuth 2.0 / OIDC authorization-code flow for
// Google, Facebook, and Apple. A provider is enabled only when its client
// credentials are configured — /v1/auth/providers tells the sites which
// buttons to show, so nothing renders that can't work.
//
// Flow: the app links to /v1/auth/oauth/{provider}/start → 302 to the
// provider → callback exchanges the code server-side, resolves the identity
// to an account (libs/user-accounts links by verified email), mints our
// session JWT, and 302s back to the app as /auth/callback#token=… (fragment,
// so the token stays out of server logs). Errors land on /login?error=….
//
// The id_token is decoded without JWKS verification: it is received directly
// from the provider's token endpoint over TLS, never from the client.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lee5i3/tcg-api/libs/httpapi"
)

// OAuthProvider is one configured identity provider.
type OAuthProvider struct {
	ID           string // "google", "facebook", "apple" — used in URLs
	Label        string // "Google" — shown on the button
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	Scopes       string
	UserInfoURL  string // set for Facebook (graph lookup); empty → identity from id_token
	FormPost     bool   // Apple: response_mode=form_post (its callback arrives as POST)
}

// identity is what a provider tells us about the signed-in person.
type identity struct {
	Subject string
	Email   string
	Name    string
}

// LoadProvidersFromEnv builds the enabled provider set from environment
// variables ({GOOGLE,FACEBOOK,APPLE}_CLIENT_ID/_CLIENT_SECRET).
func LoadProvidersFromEnv(getenv func(string) string) map[string]*OAuthProvider {
	providers := map[string]*OAuthProvider{}
	if id, secret := getenv("GOOGLE_CLIENT_ID"), getenv("GOOGLE_CLIENT_SECRET"); id != "" && secret != "" {
		providers["google"] = &OAuthProvider{
			ID: "google", Label: "Google", ClientID: id, ClientSecret: secret,
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
			Scopes:   "openid email profile",
		}
	}
	if id, secret := getenv("FACEBOOK_CLIENT_ID"), getenv("FACEBOOK_CLIENT_SECRET"); id != "" && secret != "" {
		providers["facebook"] = &OAuthProvider{
			ID: "facebook", Label: "Facebook", ClientID: id, ClientSecret: secret,
			AuthURL:     "https://www.facebook.com/v19.0/dialog/oauth",
			TokenURL:    "https://graph.facebook.com/v19.0/oauth/access_token",
			UserInfoURL: "https://graph.facebook.com/v19.0/me",
			Scopes:      "email public_profile",
		}
	}
	if id, secret := getenv("APPLE_CLIENT_ID"), getenv("APPLE_CLIENT_SECRET"); id != "" && secret != "" {
		// APPLE_CLIENT_SECRET is the pre-generated ES256 client-secret JWT
		// (Apple's scheme); rotate it before it expires (max 6 months).
		providers["apple"] = &OAuthProvider{
			ID: "apple", Label: "Apple", ClientID: id, ClientSecret: secret,
			AuthURL:  "https://appleid.apple.com/auth/authorize",
			TokenURL: "https://appleid.apple.com/auth/token",
			Scopes:   "name email",
			FormPost: true,
		}
	}
	return providers
}

// OAuth carries everything the social-login handlers need.
type OAuth struct {
	Providers map[string]*OAuthProvider
	AppURL    string // public base URL of the app site (redirect_uri host + final destination)
	Client    *http.Client
}

func (o *OAuth) httpClient() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (o *OAuth) redirectURI(provider string) string {
	return strings.TrimSuffix(o.AppURL, "/") + "/v1/auth/oauth/" + provider + "/callback"
}

// sanitizeReturnPath keeps post-login redirects inside the app.
func sanitizeReturnPath(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/"
	}
	return raw
}

// providers lists the enabled social providers (public).
func (a *API) providers(ctx context.Context, _ httpapi.Request) (httpapi.Response, error) {
	type entry struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	list := []entry{}
	for _, p := range a.OAuth.Providers {
		list = append(list, entry{ID: p.ID, Label: p.Label})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return httpapi.JSON(200, map[string]any{"providers": list})
}

// oauthStart sends the browser to the provider's consent screen.
func (a *API) oauthStart(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	provider, ok := a.OAuth.Providers[req.PathParameters["provider"]]
	if !ok {
		return httpapi.JSON(404, map[string]string{"error": "unknown or unconfigured provider"})
	}
	state, err := a.Tokens.MintState(provider.ID, sanitizeReturnPath(req.QueryStringParameters["redirect"]))
	if err != nil {
		return httpapi.Error(err)
	}
	q := url.Values{
		"client_id":     {provider.ClientID},
		"redirect_uri":  {a.OAuth.redirectURI(provider.ID)},
		"response_type": {"code"},
		"scope":         {provider.Scopes},
		"state":         {state},
	}
	if provider.FormPost {
		q.Set("response_mode", "form_post")
	}
	return httpapi.Redirect(provider.AuthURL + "?" + q.Encode())
}

// callbackParams normalizes the callback inputs: query params on GET,
// form-encoded body on POST (Apple's form_post).
func callbackParams(req httpapi.Request) url.Values {
	if strings.HasPrefix(req.RouteKey, "POST ") {
		if vals, err := url.ParseQuery(req.Body); err == nil {
			return vals
		}
		return url.Values{}
	}
	vals := url.Values{}
	for k, v := range req.QueryStringParameters {
		vals.Set(k, v)
	}
	return vals
}

// loginError sends the user back to the app's login page with a message.
func (a *API) loginError(msg string) (httpapi.Response, error) {
	return httpapi.Redirect(strings.TrimSuffix(a.OAuth.AppURL, "/") + "/login?error=" + url.QueryEscape(msg))
}

// oauthCallback finishes the flow: code → tokens → identity → our session.
func (a *API) oauthCallback(ctx context.Context, req httpapi.Request) (httpapi.Response, error) {
	provider, ok := a.OAuth.Providers[req.PathParameters["provider"]]
	if !ok {
		return httpapi.JSON(404, map[string]string{"error": "unknown or unconfigured provider"})
	}
	params := callbackParams(req)
	if errCode := params.Get("error"); errCode != "" {
		// The user cancelled at the provider (or the provider refused).
		return a.loginError("sign-in was cancelled")
	}
	stateProvider, returnPath, err := a.Tokens.VerifyState(params.Get("state"))
	if err != nil || stateProvider != provider.ID {
		return a.loginError("sign-in expired, please try again")
	}
	code := params.Get("code")
	if code == "" {
		return a.loginError("sign-in failed, please try again")
	}

	ident, err := a.exchange(ctx, provider, code)
	if err != nil {
		a.Log.Error("oauth exchange failed", "provider", provider.ID, "err", err)
		return a.loginError("sign-in with " + provider.Label + " failed")
	}
	if ident.Email == "" {
		return a.loginError(provider.Label + " did not share an email address")
	}
	user, err := a.Store.FindOrCreateOAuthUser(ctx, provider.ID, ident.Subject, ident.Email, ident.Name)
	if err != nil {
		a.Log.Error("oauth account resolve failed", "provider", provider.ID, "err", err)
		return a.loginError("sign-in with " + provider.Label + " failed")
	}
	session, err := a.Tokens.Mint(user)
	if err != nil {
		return httpapi.Error(err)
	}
	// Fragment keeps the token out of access logs; the app's /auth/callback
	// page stores it and forwards to the original destination.
	dest := strings.TrimSuffix(a.OAuth.AppURL, "/") + "/auth/callback#token=" +
		url.QueryEscape(session) + "&redirect=" + url.QueryEscape(returnPath)
	return httpapi.Redirect(dest)
}

// exchange trades the authorization code for the provider's tokens and
// extracts the user identity.
func (a *API) exchange(ctx context.Context, p *OAuthProvider, code string) (identity, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
		"redirect_uri":  {a.OAuth.redirectURI(p.ID)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := a.OAuth.httpClient().Do(req)
	if err != nil {
		return identity{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return identity{}, fmt.Errorf("token endpoint returned %d: %s", res.StatusCode, body)
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tokens); err != nil {
		return identity{}, fmt.Errorf("token endpoint body: %w", err)
	}

	if p.UserInfoURL != "" {
		return a.fetchGraphIdentity(ctx, p, tokens.AccessToken)
	}
	return identityFromIDToken(tokens.IDToken)
}

// fetchGraphIdentity is the Facebook path: the identity comes from the
// Graph API, not an id_token.
func (a *API) fetchGraphIdentity(ctx context.Context, p *OAuthProvider, accessToken string) (identity, error) {
	u := p.UserInfoURL + "?fields=id,name,email&access_token=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return identity{}, err
	}
	res, err := a.OAuth.httpClient().Do(req)
	if err != nil {
		return identity{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return identity{}, fmt.Errorf("userinfo endpoint returned %d", res.StatusCode)
	}
	var info struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return identity{}, err
	}
	return identity{Subject: info.ID, Email: info.Email, Name: info.Name}, nil
}

// identityFromIDToken decodes the OIDC id_token payload (Google, Apple).
func identityFromIDToken(idToken string) (identity, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return identity{}, fmt.Errorf("malformed id_token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return identity{}, fmt.Errorf("id_token payload: %w", err)
	}
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return identity{}, fmt.Errorf("id_token claims: %w", err)
	}
	if claims.Sub == "" {
		return identity{}, fmt.Errorf("id_token missing subject")
	}
	return identity{Subject: claims.Sub, Email: claims.Email, Name: claims.Name}, nil
}
