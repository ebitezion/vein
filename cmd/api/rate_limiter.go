package main

import (
	"context"
	"sync"
	"time"
)

type tokenBucket struct {
	capacity   float64
	tokens     float64
	refillRate float64
	updatedAt  time.Time
	mu         sync.Mutex
}

func newTokenBucket(rps float64, burst int) *tokenBucket {
	capacity := float64(burst)
	if capacity <= 0 {
		capacity = 1
	}
	if rps <= 0 {
		rps = 1
	}
	return &tokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: rps,
		updatedAt:  time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.updatedAt).Seconds()
	tb.updatedAt = now

	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	if tb.tokens < 1 {
		return false
	}

	tb.tokens--
	return true
}

type memoryRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*tokenBucket
}

func newMemoryRateLimiter() *memoryRateLimiter {
	return &memoryRateLimiter{limiters: make(map[string]*tokenBucket)}
}

func (rl *memoryRateLimiter) Allow(ctx context.Context, key string, rps float64, burst int) (bool, error) {
	rl.mu.Lock()
	limiter, found := rl.limiters[key]
	if !found {
		limiter = newTokenBucket(rps, burst)
		rl.limiters[key] = limiter
	}
	rl.mu.Unlock()

	return limiter.allow(), nil
}
