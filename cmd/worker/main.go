package main

import (
	"codepulse/internal/config"
	"codepulse/internal/queue/kafka"
	"codepulse/internal/runner"
	"codepulse/internal/storage"
	"codepulse/internal/worker"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Postgres driver
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("Error occurred while starting the worker", "error", err)
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

	taskStorage := storage.NewTaskStorage(db)
	subStorage := storage.NewSubmissionStorage(db)

	leaderboardStorage := storage.NewLeaderboardStorage(rdb)

	dockerRunner, err := runner.NewDockerRunner()
	if err != nil {
		return fmt.Errorf("docker runner initialization error: %w", err)
	}
	defer func() {
		if err := dockerRunner.Close(); err != nil {
			slog.Error("error closing docker runner", "error", err)
		}
	}()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopicSubmissions, "codepulse-workers")
	defer func() {
		if err := consumer.Close(); err != nil {
			slog.Error("error closing Kafka Consumer", "error", err)
		}
	}()

	proc := worker.NewProcessor(taskStorage, subStorage, dockerRunner, leaderboardStorage)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Worker started, waiting for submissions...")

	for {
		msg, err := consumer.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			slog.Error("failed to fetch message", "error", err)
			continue
		}

		if err := proc.Process(ctx, msg); err != nil {
			slog.Error("failed to process submission", "error", err)
			continue
		}

		if err := consumer.CommitMessage(ctx, msg); err != nil {
			slog.Error("failed to commit message", "error", err)
		}
	}
	slog.Info("Worker stopped gracefully")
	return nil
}
