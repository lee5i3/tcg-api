package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lee5i3/tcg-api/libs/httpapi"
	useraccounts "github.com/lee5i3/tcg-api/libs/user-accounts"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	client, table, err := httpapi.NewDynamoClient(context.Background())
	if err != nil {
		log.Error("startup", "err", err)
		os.Exit(1)
	}
	secret := os.Getenv("USER_JWT_SECRET")
	if secret == "" {
		log.Error("startup", "err", "USER_JWT_SECRET is not set")
		os.Exit(1)
	}
	providers := LoadProvidersFromEnv(os.Getenv)
	appURL := os.Getenv("APP_URL")
	if len(providers) > 0 && appURL == "" {
		log.Error("startup", "err", "APP_URL must be set when social providers are configured")
		os.Exit(1)
	}
	api := &API{
		Store:  useraccounts.New(client, table),
		Tokens: NewJWTTokens(secret),
		Token:  os.Getenv("API_TOKEN"),
		OAuth:  &OAuth{Providers: providers, AppURL: appURL},
		Log:    log,
	}
	lambda.Start(api.Handler())
}
