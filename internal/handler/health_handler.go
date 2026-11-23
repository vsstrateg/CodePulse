package handler

import (
	"codepulse/internal/domain"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type HealthHandler struct {
	db          *sql.DB
	rdb         *redis.Client
	kafkaBroker string
}

func NewHealthHandler(db *sql.DB, rdb *redis.Client, kafkaBroker string) *HealthHandler {
	return &HealthHandler{
		db:          db,
		rdb:         rdb,
		kafkaBroker: kafkaBroker,
	}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Warn("failed to write liveness response", "error", err)
	}
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := http.StatusOK
	verification := map[string]string{
		"postgres": "ok",
		"redis":    "ok",
		"kafka":    "ok",
	}

	if err := h.db.PingContext(ctx); err != nil {
		verification["postgres"] = err.Error()
		status = http.StatusServiceUnavailable
	}

	if err := h.rdb.Ping(ctx).Err(); err != nil {
		verification["redis"] = err.Error()
		status = http.StatusServiceUnavailable
	}

	conn, err := kafka.DialContext(ctx, "tcp", h.kafkaBroker)
	if err != nil {
		verification["kafka"] = err.Error()
		status = http.StatusServiceUnavailable
	} else {
		defer func() {
			if err := conn.Close(); err != nil {
				return
			}
		}()
	}

	respStatus := "ready"
	if status != http.StatusOK {
		respStatus = "not_ready"
	}

	resp := domain.KubernetesStatus{
		Status:       respStatus,
		Verification: verification,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Warn("failed to encode Readiness status", "error", err)
	}
}
