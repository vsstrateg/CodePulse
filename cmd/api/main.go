package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"codepulse/internal/config"
	"codepulse/internal/storage"

	// Postgres driver
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("Error occurred while starting the api", "error", err)
		os.Exit(1)
	}

}

func run() error {
	cfg := config.LoadConfig()

	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// POSTGRES
	db, err := storage.NewPostgresDB(initCtx, cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("postgres initialization error: %w", err)
	}
	slog.Info("Postgres initialized")
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("error closing DB", "error", err)
		}
	}()

	if err := storage.RunMigrations(db); err != nil {
		return fmt.Errorf("database migration error: %w", err)
	}

	return nil
}
