package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lee5i3/tcg-api/libs/httpapi"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cat, err := httpapi.NewCatalog(context.Background())
	if err != nil {
		log.Error("startup", "err", err)
		os.Exit(1)
	}
	feedURL := os.Getenv("PRICE_API_URL")
	if feedURL == "" {
		log.Error("startup", "err", ErrNoSource)
		os.Exit(1)
	}
	gameKey := os.Getenv("GAME_KEY")
	if gameKey == "" {
		gameKey = "pokemon"
	}
	job := &Job{
		Store:   cat,
		Source:  NewHTTPPriceSource(feedURL),
		GameKey: gameKey,
		Log:     log,
	}
	lambda.Start(func(ctx context.Context) (Result, error) {
		res, err := job.Run(ctx)
		log.Info("run finished", "listed", res.CardsListed, "updated", res.CardsUpdated, "failed", res.CardsFailed, "err", err)
		return res, err
	})
}
