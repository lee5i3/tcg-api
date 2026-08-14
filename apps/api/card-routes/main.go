package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

func main() {
	cat, err := httpapi.NewCatalog(context.Background())
	if err != nil {
		slog.Error("startup", "err", err)
		os.Exit(1)
	}
	api := &API{Store: cat, Token: os.Getenv("API_TOKEN")}
	lambda.Start(api.Handler())
}
