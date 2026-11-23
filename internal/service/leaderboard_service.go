package service

import (
	"codepulse/internal/domain"
	"context"
)

type LeaderboardRepository interface {
	GetTop(ctx context.Context, limit int64) ([]domain.LeaderboardEntry, error)
}

type LeaderboardService struct {
	repo LeaderboardRepository
}

func NewLeaderboardService(repo LeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{repo: repo}
}

func (s *LeaderboardService) GetTop(ctx context.Context, limit int64) ([]domain.LeaderboardEntry, error) {
	return s.repo.GetTop(ctx, limit)
}
