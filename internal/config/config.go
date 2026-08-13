// Package config loads runtime configuration from the environment, with a
// tiny .env loader for local development (real deployments set env vars).
package config

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	// Postgres connection string, e.g. postgres://user:pass@localhost:5432/db
	DatabaseURL string
	// Address the public HTTP server (GraphQL) listens on, e.g. ":8080"
	Addr string
	// Address the gRPC server (service-to-service) listens on, e.g. ":50051"
	GRPCAddr string
	// Optional bearer token; when set, every gRPC call (except reflection)
	// requires "authorization: Bearer <token>" metadata. GraphQL is public
	// read-only and never requires auth.
	APIToken string
}

var envLine = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"?([^"#]*)"?\s*(#.*)?$`)

// loadDotEnv fills os environment from ./.env for keys not already set.
func loadDotEnv() {
	raw, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		m := envLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[1], strings.TrimSpace(m[2])
		if os.Getenv(key) == "" && val != "" {
			os.Setenv(key, val)
		}
	}
}

func Load() (*Config, error) {
	loadDotEnv()

	c := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Addr:        ":" + envOr("PORT", "8080"),
		GRPCAddr:    ":" + envOr("GRPC_PORT", "50051"),
		APIToken:    os.Getenv("API_TOKEN"),
	}
	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
