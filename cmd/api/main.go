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
	"codepulse/internal/queue/kafka"
	"codepulse/internal/service"
	"codepulse/internal/storage"

	// Postgres driver
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
	// REDIS
	rdb, err := storage.NewRedisClient(initCtx, cfg.RedisAddr)
	if err != nil {
		return fmt.Errorf("redis initialization error: %w", err)
	}
	slog.Info("Redis initialized")
	defer func() {
		if err := rdb.Close(); err != nil {
			slog.Error("error closing redis", "error", err)
		}
	}()

	if err := storage.RunMigrations(db); err != nil {
		return fmt.Errorf("database migration error: %w", err)
	}

	if err := kafka.EnsureTopic(cfg.KafkaBrokers, cfg.KafkaTopicSubmissions, 4); err != nil {
		return fmt.Errorf("failed to ensure kafka topic: %w", err)
	}
	slog.Info("Kafka topic ready", "topic", cfg.KafkaTopicSubmissions)

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicSubmissions)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			slog.Error("error closing Kafka producer", "error", err)
		}
	}()

	taskStorage := storage.NewTaskStorage(db)
	subStorage := storage.NewSubmissionStorage(db)
	rateStorage := storage.NewRedisRateLimiter(rdb)
	leaderboardStorage := storage.NewLeaderboardStorage(rdb)

	taskService := service.NewTaskService(taskStorage)
	subService := service.NewSubmissionService(subStorage, taskStorage, kafkaProducer, rateStorage)
	leaderboardService := service.NewLeaderboardService(leaderboardStorage)

	taskHandler := handler.NewTaskHandler(taskService)
	subHandler := handler.NewSubmissionHandler(subService)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)
	healthHandler := handler.NewHealthHandler(db, rdb, cfg.KafkaBrokers)

	mux := http.NewServeMux()
	healthMux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/tasks", taskHandler.CreateTask)
	mux.HandleFunc("GET /api/v1/tasks", taskHandler.GetTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}", taskHandler.GetTaskByID)

	mux.HandleFunc("POST /api/v1/submissions", subHandler.CreateSubmission)
	mux.HandleFunc("GET /api/v1/submissions/{id}", subHandler.GetSubmissionByID)
	mux.HandleFunc("GET /api/v1/users/{user_id}/submissions", subHandler.GetUserSubmissions)

	mux.HandleFunc("GET /api/v1/leaderboard", leaderboardHandler.GetLeaderboard)

	healthMux.HandleFunc("GET /healthz", healthHandler.Liveness)
	healthMux.HandleFunc("GET /ready", healthHandler.Readiness)

	srv := &http.Server{
		Addr:    cfg.Port,
		Handler: mux,
	}

	healthSrv := &http.Server{
		Addr:    cfg.HealthPort,
		Handler: healthMux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
		}
	}()
	slog.Info("Server started", "port", cfg.Port)

	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Health server error", "error", err)
		}
	}()
	slog.Info("Health server started", "port", cfg.HealthPort)

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

	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error while stopping the health server", "error", err)
	}
	slog.Info("The Health server has been shut down successfully and gracefully")

	return nil
}
