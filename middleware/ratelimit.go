package middleware

import (
	"sync"
	"time"

	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// visitor tracks the request count and window for a single IP.
type visitor struct {
	count    int
	windowStart time.Time
}

// RateLimiter is an in-memory, per-IP rate limiter using a sliding window.
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	limit    int           // max requests per window
	window   time.Duration // time window
}

// NewRateLimiter creates a new rate limiter with the given limit per minute.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}

	// Start a background goroutine to clean up expired entries every 5 minutes
	go rl.cleanup()

	return rl
}

// Middleware returns a Gin middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.allow(ip) {
			utils.TooManyRequests(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow checks if a request from the given IP is allowed.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	v, exists := rl.visitors[ip]
	if !exists || now.Sub(v.windowStart) > rl.window {
		// New visitor or window expired — reset
		rl.visitors[ip] = &visitor{
			count:    1,
			windowStart: now,
		}
		return true
	}

	// Within the current window
	if v.count >= rl.limit {
		return false
	}

	v.count++
	return true
}

// cleanup periodically removes expired visitor entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.windowStart) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
