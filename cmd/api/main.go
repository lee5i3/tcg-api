// The TCG API server — a standalone trading-card catalog.
//
// Two surfaces share one catalog:
//   - gRPC (:50051)  — trusted service-to-service read/write
//   - GraphQL (:8080/graphql) — public read-only queries, with an embedded
//     GraphiQL playground at /graphiql
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lee5i3/pokemon-invest/internal/catalog"
	"github.com/lee5i3/pokemon-invest/internal/config"
	"github.com/lee5i3/pokemon-invest/internal/graphql"
	"github.com/lee5i3/pokemon-invest/internal/postgres"
	"github.com/lee5i3/pokemon-invest/internal/rpc"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	cat := catalog.New(db)
	if err := cat.Seed(ctx); err != nil {
		log.Error("seed games", "err", err)
		os.Exit(1)
	}

	// gRPC — service-to-service read/write
	grpcServer := rpc.NewGRPCServer(cat, cfg.APIToken)
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("grpc listen", "err", err)
		os.Exit(1)
	}

	// HTTP — public GraphQL + playground + health
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"database unreachable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	graphql.Mount(mux, cat)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errs := make(chan error, 2)
	go func() {
		log.Info("gRPC listening", "addr", cfg.GRPCAddr, "auth", cfg.APIToken != "")
		errs <- grpcServer.Serve(grpcListener)
	}()
	go func() {
		log.Info("GraphQL listening", "addr", cfg.Addr, "path", "/graphql")
		errs <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		grpcServer.GracefulStop()
	case err := <-errs:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}
}
