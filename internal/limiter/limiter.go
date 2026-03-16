package limiter

import (
	"context"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type Tier string

const (
	VIP    Tier = "VIP"
	Normal Tier = "Normal"
)

// RateLimiter manages rate limiting for users based on their tier.
// In a full implementation, this uses a distributed rate limiter via Redis.
type RateLimiter struct {
	rdb *redis.Client

	// Fallback in-memory limiters for simplicity in this phase
	limits map[string]*rate.Limiter
}

func NewRateLimiter(redisAddr string) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &RateLimiter{
		rdb:    rdb,
		limits: make(map[string]*rate.Limiter),
	}
}

// Allow checks if the given userID is allowed to make a request based on their tier.
func (rl *RateLimiter) Allow(ctx context.Context, userID string, tier Tier) bool {
	// A simple in-memory simulation of Redis-based token bucket.
	// VIP users get 10 req/sec, burst of 20.
	// Normal users get 2 req/sec, burst of 5.

	limiter, exists := rl.limits[userID]
	if !exists {
		if tier == VIP {
			limiter = rate.NewLimiter(rate.Limit(10), 20)
		} else {
			limiter = rate.NewLimiter(rate.Limit(2), 5)
		}
		rl.limits[userID] = limiter
	}

	return limiter.Allow()
}
