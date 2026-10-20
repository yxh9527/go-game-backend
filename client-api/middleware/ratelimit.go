package middleware

import (
	"net/http"
	"sync"
	"time"

	"client-api/common"

	"github.com/gin-gonic/gin"
)

type ipBucket struct {
	tokens   float64
	lastTime time.Time
}

// RateLimiter 进程内令牌桶限流（按 IP）。
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*ipBucket
	rate     float64 // tokens per second
	burst    float64
	cleanAt  time.Time
}

func NewRateLimiter(qps, burst int) *RateLimiter {
	if qps <= 0 {
		qps = 20
	}
	if burst <= 0 {
		burst = qps * 2
	}
	rl := &RateLimiter{
		buckets: make(map[string]*ipBucket),
		rate:    float64(qps),
		burst:   float64(burst),
		cleanAt: time.Now().Add(time.Minute),
	}
	return rl
}

func (r *RateLimiter) allow(ip string) bool {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()

	if now.After(r.cleanAt) {
		for k, b := range r.buckets {
			if now.Sub(b.lastTime) > 2*time.Minute {
				delete(r.buckets, k)
			}
		}
		r.cleanAt = now.Add(time.Minute)
	}

	b := r.buckets[ip]
	if b == nil {
		r.buckets[ip] = &ipBucket{tokens: r.burst - 1, lastTime: now}
		return true
	}
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * r.rate
	if b.tokens > r.burst {
		b.tokens = r.burst
	}
	b.lastTime = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !r.allow(ip) {
			common.Fail(c, http.StatusOK, 429, "请求过于频繁")
			c.Abort()
			return
		}
		c.Next()
	}
}
