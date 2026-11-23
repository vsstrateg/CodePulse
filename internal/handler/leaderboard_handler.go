package handler

import (
	"codepulse/internal/domain"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type LeaderboardService interface {
	GetTop(ctx context.Context, limit int64) ([]domain.LeaderboardEntry, error)
}
type LeaderboardHandler struct {
	service LeaderboardService
}

func NewLeaderboardHandler(service LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{service: service}
}

func (h *LeaderboardHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := int64(10)

	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, err := h.service.GetTop(r.Context(), limit)
	if err != nil {
		slog.Error("failed to get leaderboard", "error", err)
		sendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		slog.Warn("failed to encode leaderboard response", "error", err)
	}
}
