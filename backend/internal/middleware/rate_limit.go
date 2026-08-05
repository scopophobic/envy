package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitKeyFunc func(*gin.Context) string

type rateLimitBucket struct {
	count   int
	resetAt time.Time
}

// RateLimiter is a process-local fixed-window limiter. It protects individual
// API instances; deployments with multiple backend replicas should replace it
// with a shared Redis-backed implementation while keeping the same middleware.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateLimitBucket
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]rateLimitBucket),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

func ClientIPRateLimitKey(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}
	return fmt.Sprintf("ip:%s:%s:%s", c.ClientIP(), c.Request.Method, route)
}

func AuthenticatedRateLimitKey(c *gin.Context) string {
	if userID, err := GetCurrentUserID(c); err == nil {
		return fmt.Sprintf("user:%s:%s:%s", userID, c.Request.Method, c.FullPath())
	}
	return ClientIPRateLimitKey(c)
}

func (l *RateLimiter) Middleware(keyFn RateLimitKeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || l.limit <= 0 || l.window <= 0 {
			c.Next()
			return
		}
		keySelector := keyFn
		if keySelector == nil {
			keySelector = ClientIPRateLimitKey
		}

		now := l.now()
		key := keySelector(c)

		l.mu.Lock()
		bucket, exists := l.buckets[key]
		if !exists || !now.Before(bucket.resetAt) {
			bucket = rateLimitBucket{resetAt: now.Add(l.window)}
		}

		allowed := bucket.count < l.limit
		if allowed {
			bucket.count++
		}
		l.buckets[key] = bucket
		if len(l.buckets) > 4096 {
			for bucketKey, candidate := range l.buckets {
				if !now.Before(candidate.resetAt) {
					delete(l.buckets, bucketKey)
				}
			}
		}
		l.mu.Unlock()

		remaining := l.limit - bucket.count
		if remaining < 0 {
			remaining = 0
		}
		resetSeconds := int(bucket.resetAt.Sub(now).Seconds())
		if resetSeconds < 1 {
			resetSeconds = 1
		}
		c.Header("RateLimit-Limit", strconv.Itoa(l.limit))
		c.Header("RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("RateLimit-Reset", strconv.FormatInt(bucket.resetAt.Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(resetSeconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests. Please retry later.",
				"retry_after": resetSeconds,
			})
			return
		}

		c.Next()
	}
}
