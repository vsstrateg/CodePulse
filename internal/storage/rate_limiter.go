package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
	}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	now := time.Now().UnixNano()

	clearBefore := now - window.Nanoseconds()

	if err := r.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(clearBefore, 10)).Err(); err != nil {
		return false, err
	}

	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count >= limit {
		return false, nil
	}

	err = r.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: strconv.FormatInt(now, 10),
	}).Err()
	if err != nil {
		return false, err
	}

	if err := r.client.Expire(ctx, key, window).Err(); err != nil {
		return false, err
	}

	return true, nil
}
