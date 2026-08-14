package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	useraccounts "github.com/lee5i3/tcg-api/libs/user-accounts"
)

// tokenTTL is how long an app session lasts before the user signs in again.
const tokenTTL = 7 * 24 * time.Hour

// JWTTokens mints and verifies the app's user session tokens (HS256).
type JWTTokens struct {
	Secret []byte
	now    func() time.Time
}

func NewJWTTokens(secret string) *JWTTokens {
	return &JWTTokens{Secret: []byte(secret), now: time.Now}
}

// Mint issues a signed session token for a user.
func (t *JWTTokens) Mint(user useraccounts.User) (string, error) {
	now := t.now()
	claims := jwt.RegisteredClaims{
		Subject:   user.ID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		Issuer:    "tcg-api-auth",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.Secret)
}

// ErrBadToken is returned for missing, malformed, or expired session tokens.
var ErrBadToken = errors.New("invalid or expired session")

// stateTTL bounds how long an OAuth round-trip may take.
const stateTTL = 10 * time.Minute

const stateIssuer = "tcg-api-oauth-state"

type stateClaims struct {
	jwt.RegisteredClaims
	Provider string `json:"prv"`
	Redirect string `json:"rdr"`
}

// MintState issues the signed CSRF state for an OAuth round-trip. It carries
// no subject, so it can never pass session verification.
func (t *JWTTokens) MintState(provider, redirect string) (string, error) {
	now := t.now()
	claims := stateClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    stateIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(stateTTL)),
		},
		Provider: provider,
		Redirect: redirect,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.Secret)
}

// VerifyState checks an OAuth state token and returns the provider and the
// post-login redirect path it was minted for.
func (t *JWTTokens) VerifyState(raw string) (provider, redirect string, err error) {
	parsed, err := jwt.ParseWithClaims(raw, &stateClaims{}, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", tok.Header["alg"])
		}
		return t.Secret, nil
	}, jwt.WithTimeFunc(func() time.Time { return t.now() }), jwt.WithIssuer(stateIssuer))
	if err != nil || !parsed.Valid {
		return "", "", ErrBadToken
	}
	claims, ok := parsed.Claims.(*stateClaims)
	if !ok || claims.Provider == "" {
		return "", "", ErrBadToken
	}
	return claims.Provider, claims.Redirect, nil
}

// Verify checks a session token and returns the user id it belongs to.
func (t *JWTTokens) Verify(raw string) (string, error) {
	parsed, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", tok.Header["alg"])
		}
		return t.Secret, nil
	}, jwt.WithTimeFunc(func() time.Time { return t.now() }))
	if err != nil || !parsed.Valid {
		return "", ErrBadToken
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", ErrBadToken
	}
	return claims.Subject, nil
}
