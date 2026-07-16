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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker/v2"

	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"
	"github.com/shahadulhaider/verso/libs/go/logger"
	versootel "github.com/shahadulhaider/verso/libs/go/otel"

	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/config"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/consumer"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/service"
)

const svcName = "verso-profile-service"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(svcName)
	cfg := config.Load()

	shutdown, err := versootel.Init(ctx, svcName)
	if err != nil {
		log.Warn("otel init failed, continuing without tracing", slog.String("error", err.Error()))
	} else {
		defer shutdown(ctx)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("connected to database")

	jwtValidator := versojwt.NewJWKSValidator(cfg.JWKSURL)
	if err := jwtValidator.Start(ctx); err != nil {
		log.Warn("jwks init failed, auth will reject requests until identity service is available",
			slog.String("error", err.Error()))
	}

	cb := gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
		Name:        svcName,
		Timeout:     30 * time.Second,
		MaxRequests: 5,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Info("circuit breaker state change",
				slog.String("name", name),
				slog.String("from", from.String()),
				slog.String("to", to.String()))
		},
	})

	repo := repository.New(pool)
	svc := service.New(repo, log)
	h := handler.New(svc, repo, cb)

	cons, err := consumer.New(cfg.RedpandaBrokers, svc, log)
	if err != nil {
		log.Error("init kafka consumer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer cons.Close()

	go cons.Run(ctx)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)

	r.Group(func(r chi.Router) {
		r.Use(versojwt.Middleware(jwtValidator))
		r.Get("/v1/profiles/me", h.GetMyProfile)
		r.Patch("/v1/profiles/me", h.UpdateProfile)
	})

	r.Get("/v1/profiles/{userId}", h.GetProfile)

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
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)
}

func runHealthcheck() {
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8004"
	}
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
