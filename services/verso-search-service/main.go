package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/shahadulhaider/verso/libs/go/logger"
	versootel "github.com/shahadulhaider/verso/libs/go/otel"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/config"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/indexer"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
)

const serviceName = "verso-search-service"

func main() {
	// Healthcheck subcommand for Docker compose.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(serviceName)
	cfg := config.Load()

	shutdown, err := versootel.Init(ctx, serviceName)
	if err != nil {
		log.Warn("otel init failed, continuing without tracing", slog.String("error", err.Error()))
	} else {
		defer shutdown(ctx)
	}

	// OpenSearch client — retry connection with backoff.
	osClient := opensearch.New(cfg.OpenSearchURL, log)
	var osReady bool
	for attempt := 1; attempt <= 10; attempt++ {
		if err := osClient.EnsureIndex(ctx); err != nil {
			log.Warn("opensearch not ready, retrying...",
				slog.String("error", err.Error()),
				slog.Int("attempt", attempt))
			time.Sleep(3 * time.Second)
			continue
		}
		osReady = true
		break
	}
	if !osReady {
		log.Error("opensearch unavailable after 10 retries")
		os.Exit(1)
	}

	// Kafka consumer / indexer.
	ix, err := indexer.New(cfg.RedpandaBrokers, osClient, log)
	if err != nil {
		log.Error("init indexer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer ix.Close()

	go ix.Run(ctx)

	// HTTP server.
	h := handler.New(osClient)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Get("/v1/search", h.Search)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("starting server", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	cancel() // stop the indexer consumer loop

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)
}

func runHealthcheck() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
