package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codepulse/internal/config"
	"codepulse/internal/handler"
	"codepulse/internal/service"
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

	taskStorage := storage.NewTaskStorage(db)
	taskService := service.NewTaskService(taskStorage)
	taskHandler := handler.NewTaskHandler(taskService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/tasks", taskHandler.CreateTask)
	mux.HandleFunc("GET /api/v1/tasks", taskHandler.GetTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}", taskHandler.GetTaskByID)

	srv := &http.Server{
		Addr:    cfg.Port,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
		}
	}()
	slog.Info("Server started", "port", cfg.Port)

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-stopCtx.Done()
	slog.Info("Stop signal received, shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error while forcibly stopping the server", "error", err)
	}
	slog.Info("The server has been shut down successfully and gracefully")

	return nil
}
