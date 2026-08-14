package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

func main() {
	ctx := context.Background()
	cat, err := httpapi.NewCatalog(ctx)
	if err != nil {
		slog.Error("startup", "err", err)
		os.Exit(1)
	}
	// Seed the default games on cold start, like the old server did at boot.
	if err := cat.Seed(ctx); err != nil {
		slog.Error("seed games", "err", err)
		os.Exit(1)
	}
	api := &API{Store: cat, Token: os.Getenv("API_TOKEN")}
	lambda.Start(api.Handler())
}
