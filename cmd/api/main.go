package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kundanmergu/cloud-search-engine/backend/internal/api"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/config"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/retrieval"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := storage.ConnectPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := storage.New(pool)
	searcher := retrieval.NewSearcher(
		store,
		retrieval.BM25Params{K1: cfg.BM25K1, B: cfg.BM25B},
		cfg.DefaultLimit,
		cfg.MaxLimit,
		cfg.CandidateLimit,
	)

	srv := api.NewServer(store, searcher, logger)
	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      srv.Router(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete", "at", time.Now().UTC().Format(time.RFC3339))
}
