package internal

import (
	"sync"
	"time"
)

// RateLimiter optionally simulates Upstash cloud quotas and RPS limits.
type RateLimiter struct {
	quotaLimit int64
	rpsLimit   int

	requestCount atomicInt64
	windowStart  time.Time
	windowCount  int
	mu           sync.Mutex
}

type atomicInt64 struct {
	v int64
	mu sync.Mutex
}

func (a *atomicInt64) Add(n int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.v += n
	return a.v
}

func (a *atomicInt64) Load() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.v
}

func NewRateLimiter(quotaLimit int64, rpsLimit int) *RateLimiter {
	return &RateLimiter{
		quotaLimit:  quotaLimit,
		rpsLimit:    rpsLimit,
		windowStart: time.Now(),
	}
}

func (r *RateLimiter) Allow() (bool, string) {
	if r == nil {
		return true, ""
	}

	if r.quotaLimit > 0 {
		count := r.requestCount.Add(1)
		if count > r.quotaLimit {
			return false, "ERR daily request quota exceeded (simulated Upstash free tier limit)"
		}
	}

	if r.rpsLimit > 0 {
		r.mu.Lock()
		now := time.Now()
		if now.Sub(r.windowStart) >= time.Second {
			r.windowStart = now
			r.windowCount = 0
		}
		r.windowCount++
		over := r.windowCount > r.rpsLimit
		r.mu.Unlock()
		if over {
			return false, "ERR rate limit exceeded (simulated Upstash RPS limit)"
		}
	}

	return true, ""
}
