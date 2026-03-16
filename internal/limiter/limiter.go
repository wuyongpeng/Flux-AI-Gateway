package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Tier string

const (
	VIP    Tier = "VIP"
	Normal Tier = "Normal"
)

// RateLimiter manages rate limiting for users using Redis.
type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(redisAddr string) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &RateLimiter{
		rdb: rdb,
	}
}

// Allow checks if the given userID is allowed to make a request based on their tier.
// It uses a simple Redis-based fixed-window or token-bucket approach.
func (rl *RateLimiter) Allow(ctx context.Context, userID string, tier Tier) bool {
	// Simple Redis implementation:
	// Key: flux:limiter:<userid>
	// Max requests per minute: VIP=600, Normal=120

	key := fmt.Sprintf("flux:limiter:%s", userID)
	limit := 120
	if tier == VIP {
		limit = 600
	}

	// EXECUTE IN REDIS: INCR key; EXPIRE key 60
	pipe := rl.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true // Fail open if Redis is down
	}

	return int(incr.Val()) <= limit
}
