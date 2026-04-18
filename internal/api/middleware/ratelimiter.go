package middleware

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	apperrors "github.com/VatsalP117/hostbox/internal/errors"
)

// TokenBucket implements a per-key token bucket.
type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimiterConfig configures a rate limiter.
type RateLimiterConfig struct {
	Rate  int // requests per minute
	Burst int // max burst (usually same as Rate)
}

// RateLimiter stores token buckets per key.
type RateLimiter struct {
	buckets sync.Map
	config  RateLimiterConfig
}

// NewRateLimiter creates a rate limiter. Starts a cleanup goroutine.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{config: cfg}
	go rl.cleanup()
	return rl
}

// Allow checks if the key is allowed a request.
func (rl *RateLimiter) Allow(key string) (bool, int, time.Time) {
	now := time.Now()
	val, _ := rl.buckets.LoadOrStore(key, &TokenBucket{
		tokens:     float64(rl.config.Burst),
		maxTokens:  float64(rl.config.Burst),
		refillRate: float64(rl.config.Rate) / 60.0,
		lastRefill: now,
	})
	bucket := val.(*TokenBucket)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	resetTime := now.Add(time.Duration(float64(time.Minute) / float64(rl.config.Rate)))

	if bucket.tokens < 1 {
		return false, 0, resetTime
	}

	bucket.tokens--
	return true, int(bucket.tokens), resetTime
}

// cleanup removes stale buckets every 10 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		threshold := time.Now().Add(-10 * time.Minute)
		rl.buckets.Range(func(key, value interface{}) bool {
			bucket := value.(*TokenBucket)
			bucket.mu.Lock()
			stale := bucket.lastRefill.Before(threshold)
			bucket.mu.Unlock()
			if stale {
				rl.buckets.Delete(key)
			}
			return true
		})
	}
}

// RateLimit returns Echo middleware using the given limiter and key extraction function.
func RateLimit(limiter *RateLimiter, keyFunc func(echo.Context) string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := keyFunc(c)
			allowed, remaining, resetTime := limiter.Allow(key)

			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.config.Rate))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Response().Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				return &apperrors.AppError{
					Code:    "RATE_LIMITED",
					Message: "Too many requests",
					Status:  429,
				}
			}

			return next(c)
		}
	}
}

// IPKeyFunc extracts client IP for rate limiting.
func IPKeyFunc(c echo.Context) string {
	return c.RealIP()
}

// UserKeyFunc extracts user ID for rate limiting, falls back to IP.
func UserKeyFunc(c echo.Context) string {
	user := GetUser(c)
	if user != nil {
		return user.ID
	}
	return c.RealIP()
}
