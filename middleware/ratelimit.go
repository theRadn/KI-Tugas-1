package middleware

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type IPRateLimiter struct {
	mu sync.Mutex

	buckets map[string]*bucket

	capacity   float64
	refillRate float64
	ttl        time.Duration

	stop chan struct{}
	once sync.Once
}

func NewIPRateLimiter(
	capacity int,
	refillTokens int,
	refillPeriod time.Duration,
	ttl time.Duration,
	cleanupInterval time.Duration,
) (*IPRateLimiter, error) {
	if capacity < 1 || refillTokens < 1 {
		return nil, fmt.Errorf("capacity and refill tokens must be positive")
	}

	if refillPeriod <= 0 || ttl <= 0 || cleanupInterval <= 0 {
		return nil, fmt.Errorf("durations must be positive")
	}

	limiter := &IPRateLimiter{
		buckets:    make(map[string]*bucket),
		capacity:   float64(capacity),
		refillRate: float64(refillTokens) / refillPeriod.Seconds(),
		ttl:        ttl,
		stop:       make(chan struct{}),
	}

	go limiter.cleanupLoop(cleanupInterval)

	return limiter, nil
}

func (l *IPRateLimiter) Allow(ip string) (bool, time.Duration) {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, exists := l.buckets[ip]

	if !exists {
		b = &bucket{
			tokens:     l.capacity,
			lastRefill: now,
			lastSeen:   now,
		}
		l.buckets[ip] = b
	} else {
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens = math.Min(
			l.capacity,
			b.tokens+elapsed*l.refillRate,
		)
		b.lastRefill = now
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	wait := time.Duration(
		math.Ceil((1 - b.tokens) / l.refillRate * float64(time.Second)),
	)

	return false, wait
}

func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		allowed, retryAfter := l.Allow(ip)

		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}

			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

func (l *IPRateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanupExpired()
		case <-l.stop:
			return
		}
	}
}

func (l *IPRateLimiter) cleanupExpired() {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for ip, b := range l.buckets {
		if now.Sub(b.lastSeen) >= l.ttl {
			delete(l.buckets, ip)
		}
	}
}

func (l *IPRateLimiter) Close() {
	l.once.Do(func() {
		close(l.stop)
	})
}
