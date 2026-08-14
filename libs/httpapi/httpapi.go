// Package httpapi is the shared plumbing for the Lambda functions behind the
// API Gateway HTTP API: JSON responses, domain-error → status mapping, and
// bearer-token auth for write routes.
//
// Route → Lambda mapping (configured in infra/terraform):
//
//	apps/api/games  GET/POST /v1/games, GET /healthz
//	apps/api/sets   GET/POST /v1/games/{game}/sets, PUT/DELETE /v1/games/{game}/sets/{set}
//	apps/api/cards  GET /v1/games/{game}/sets/{set}/cards, GET /v1/games/{game}/cards,
//	                GET/PUT/DELETE /v1/games/{game}/cards/{card}, POST /v1/games/{game}/cards
//
// Reads are public; writes (POST/PUT/DELETE) require "Authorization: Bearer
// <API_TOKEN>" when API_TOKEN is set — the same trust model the old gRPC
// surface had.
package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lee5i3/tcg-api/libs/card-catalog-store"
)

type Request = events.APIGatewayV2HTTPRequest
type Response = events.APIGatewayV2HTTPResponse

// Handler is one routed endpoint.
type Handler func(ctx context.Context, req Request) (Response, error)

// NewDynamoClient builds a DynamoDB client and table name from the
// environment: TABLE_NAME (required) and DYNAMODB_ENDPOINT (optional, for
// DynamoDB Local).
func NewDynamoClient(ctx context.Context) (*dynamodb.Client, string, error) {
	table := os.Getenv("TABLE_NAME")
	if table == "" {
		return nil, "", errors.New("TABLE_NAME is not set")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, "", err
	}
	var opts []func(*dynamodb.Options)
	if ep := os.Getenv("DYNAMODB_ENDPOINT"); ep != "" {
		opts = append(opts, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(ep) })
	}
	return dynamodb.NewFromConfig(cfg, opts...), table, nil
}

// NewCatalog builds the card-catalog store from the environment.
func NewCatalog(ctx context.Context) (*catalog.Catalog, error) {
	client, table, err := NewDynamoClient(ctx)
	if err != nil {
		return nil, err
	}
	return catalog.New(client, table), nil
}

// JSON renders a JSON body with the given status.
func JSON(status int, body any) (Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	return Response{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(raw),
	}, nil
}

// Error maps domain errors onto HTTP statuses.
func Error(err error) (Response, error) {
	status := 500
	msg := "internal error"
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		status, msg = 404, err.Error()
	case errors.Is(err, catalog.ErrInvalid):
		status, msg = 400, err.Error()
	case errors.Is(err, catalog.ErrConflict):
		status, msg = 409, err.Error()
	default:
		slog.Error("unhandled", "err", err)
	}
	return JSON(status, map[string]string{"error": msg})
}

// ErrBadRequest renders a 400 with a caller-facing message.
func ErrBadRequest(msg string) (Response, error) {
	return JSON(400, map[string]string{"error": msg})
}

// Redirect renders a 302 to the given URL.
func Redirect(url string) (Response, error) {
	return Response{
		StatusCode: 302,
		Headers:    map[string]string{"Location": url},
	}, nil
}

// Authorized checks the bearer token on write routes. An empty configured
// token disables auth (local development), matching the old gRPC behavior.
func Authorized(req Request, token string) bool {
	if token == "" {
		return true
	}
	header := req.Headers["authorization"]
	if header == "" {
		header = req.Headers["Authorization"]
	}
	got, ok := strings.CutPrefix(header, "Bearer ")
	return ok && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// Route dispatches on the API Gateway route key ("GET /v1/games"), guarding
// every non-GET route with the bearer token.
func Route(token string, routes map[string]Handler) Handler {
	return Dispatch(token, routes, nil)
}

// Dispatch routes with an explicit auth policy: protected routes require the
// bearer token on non-GET methods (the standard write guard); public routes
// never do (e.g. login/register, which exist to establish credentials).
func Dispatch(token string, protected, public map[string]Handler) Handler {
	return func(ctx context.Context, req Request) (Response, error) {
		if h, ok := public[req.RouteKey]; ok {
			return h(ctx, req)
		}
		h, ok := protected[req.RouteKey]
		if !ok {
			return JSON(404, map[string]string{"error": "no such route"})
		}
		if !strings.HasPrefix(req.RouteKey, "GET ") && !Authorized(req, token) {
			return JSON(401, map[string]string{"error": "missing or invalid bearer token"})
		}
		return h(ctx, req)
	}
}

// Body decodes a JSON request body into v.
func Body(req Request, v any) error {
	return json.Unmarshal([]byte(req.Body), v)
}
