package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"sentinelops/internal/config"
	"sentinelops/internal/httpapi"
	"sentinelops/internal/postgres"
	"sentinelops/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		response, err := http.Get("http://127.0.0.1" + cfg.ListenAddr + "/readyz")
		if err != nil {
			os.Exit(1)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	var repo store.Repository
	var db *sql.DB
	if cfg.DatabaseURL != "" {
		db, err = sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			logger.Error("open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		db.SetMaxOpenConns(15)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err = postgres.Migrate(ctx, db); err != nil {
			logger.Error("run migrations", "error", err)
			os.Exit(1)
		}
		if err = db.PingContext(ctx); err != nil {
			logger.Error("connect database", "error", err)
			os.Exit(1)
		}
		repo = postgres.New(db)
		logger.Info("database connected and migrations applied")
	} else {
		repo = store.NewMemory()
		logger.Warn("DATABASE_URL unset; using in-memory demonstration store")
	}
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: httpapi.New(repo, cfg.AllowDemoActions, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "address", cfg.ListenAddr, "demo_actions", cfg.AllowDemoActions)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			_ = srv.Close()
		}
		logger.Info("server stopped")
	}
}
