// local-gateway stands in for API Gateway during local development: it
// accepts plain HTTP, wraps each request into an API Gateway v2 proxy event,
// POSTs it to the right Lambda container's Runtime Interface Emulator
// (which the AWS base images bundle), and unwraps the response.
//
// It exists only for docker-compose — production routing lives in
// infra/terraform/apigateway.tf, and the route table here must mirror it.
package main

import (
	"log"
	"net/http"
	"os"
)

// routeTable mirrors infra/terraform/apigateway.tf locals.routes.
var routeTable = []route{
	{"GET /healthz", "game"},
	{"GET /v1/games", "game"},
	{"POST /v1/games", "game"},
	{"POST /v1/auth/register", "auth"},
	{"POST /v1/auth/login", "auth"},
	{"GET /v1/auth/me", "auth"},
	{"POST /v1/auth/check", "auth"},
	{"GET /v1/auth/providers", "auth"},
	{"GET /v1/auth/oauth/{provider}/start", "auth"},
	{"GET /v1/auth/oauth/{provider}/callback", "auth"},
	{"POST /v1/auth/oauth/{provider}/callback", "auth"},
	{"GET /v1/games/{game}/sets", "set"},
	{"POST /v1/games/{game}/sets", "set"},
	{"PUT /v1/games/{game}/sets/{set}", "set"},
	{"DELETE /v1/games/{game}/sets/{set}", "set"},
	{"GET /v1/games/{game}/sets/{set}/cards", "card"},
	{"GET /v1/games/{game}/cards", "card"},
	{"POST /v1/games/{game}/cards", "card"},
	{"GET /v1/games/{game}/cards/{card}", "card"},
	{"PUT /v1/games/{game}/cards/{card}", "card"},
	{"DELETE /v1/games/{game}/cards/{card}", "card"},
}

func main() {
	upstreams := map[string]string{
		"game": envOr("GAME_ROUTES_URL", "http://lambda-game-routes:8080"),
		"set":  envOr("SET_ROUTES_URL", "http://lambda-set-routes:8080"),
		"card": envOr("CARD_ROUTES_URL", "http://lambda-card-routes:8080"),
		"auth": envOr("AUTH_ROUTES_URL", "http://lambda-auth-routes:8080"),
	}
	addr := envOr("LISTEN_ADDR", ":8080")

	gw, err := newGateway(routeTable, upstreams)
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}
	log.Printf("local-gateway listening on %s (game=%s set=%s card=%s auth=%s)",
		addr, upstreams["game"], upstreams["set"], upstreams["card"], upstreams["auth"])
	log.Fatal(http.ListenAndServe(addr, gw))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
