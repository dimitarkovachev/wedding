package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter tracks per-IP token bucket limiters.
type RateLimiter struct {
	visitors sync.Map
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a Gin middleware that applies per-IP rate limiting.
// rps controls the steady-state rate (requests per second), burst is the
// maximum number of tokens that can be consumed in a single burst.
func NewRateLimiter(rps rate.Limit, burst int) gin.HandlerFunc {
	rl := &RateLimiter{rps: rps, burst: burst}
	go rl.cleanupLoop()
	return rl.handle
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	val, ok := rl.visitors.Load(ip)
	if ok {
		v := val.(*visitor)
		v.lastSeen = time.Now()
		return v.limiter
	}

	limiter := rate.NewLimiter(rl.rps, rl.burst)
	rl.visitors.Store(ip, &visitor{limiter: limiter, lastSeen: time.Now()})
	return limiter
}

func (rl *RateLimiter) handle(c *gin.Context) {
	ip := c.ClientIP()
	limiter := rl.getVisitor(ip)

	if !limiter.Allow() {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"message": "too many requests, please try again later",
		})
		return
	}

	c.Next()
}

// cleanupLoop removes visitors that haven't been seen for 3 minutes.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.visitors.Range(func(key, value any) bool {
			v := value.(*visitor)
			if time.Since(v.lastSeen) > 3*time.Minute {
				rl.visitors.Delete(key)
			}
			return true
		})
	}
}
