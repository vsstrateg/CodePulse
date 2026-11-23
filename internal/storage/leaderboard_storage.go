package storage

import (
	"codepulse/internal/domain"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type LeaderboardStorage struct {
	client *redis.Client
}

func NewLeaderboardStorage(client *redis.Client) *LeaderboardStorage {
	return &LeaderboardStorage{client: client}
}

func (s *LeaderboardStorage) IncrementScore(ctx context.Context, userID string, points int) error {
	return s.client.ZIncrBy(ctx, "contest:leaderboard", float64(points), userID).Err()
}

func (s *LeaderboardStorage) GetTop(ctx context.Context, limit int64) ([]domain.LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	res, err := s.client.ZRevRangeWithScores(ctx, "contest:leaderboard", 0, limit-1).Result()
	if err != nil {
		return []domain.LeaderboardEntry{}, err
	}

	entries := make([]domain.LeaderboardEntry, 0, len(res))
	for _, v := range res {
		entries = append(entries, domain.LeaderboardEntry{
			UserID: fmt.Sprint(v.Member),
			Score:  int(v.Score),
		})
	}
	return entries, nil
}
