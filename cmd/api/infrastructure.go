package main

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

var errCacheMiss = errors.New("cache miss")

type Cache interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}

type IdempotencyStore interface {
	Reserve(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, rps float64, burst int) (bool, error)
}

type JobQueue interface {
	Register(name string, handler JobHandler)
	Publish(ctx context.Context, job Job) error
	Start(ctx context.Context, workers int)
}

type infrastructure struct {
	cache       Cache
	idempotency IdempotencyStore
	rateLimiter RateLimiter
	queue       JobQueue
	cleanup     func(context.Context) error
}

func setupInfrastructure(cfg config, logger *log.Logger) (infrastructure, error) {
	if cfg.redis.enabled {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.redis.addr,
			Password: cfg.redis.password,
			DB:       cfg.redis.db,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			return infrastructure{}, err
		}

		logger.Printf("redis enabled at %s", cfg.redis.addr)
		return infrastructure{
			cache:       newRedisCache(client),
			idempotency: newRedisIdempotencyStore(client),
			rateLimiter: newRedisRateLimiter(client),
			queue:       newRedisQueue(client, cfg.redis.queueKey),
			cleanup: func(ctx context.Context) error {
				return client.Close()
			},
		}, nil
	}

	logger.Println("using in-memory cache/queue/rate-limit/idempotency backends")
	return infrastructure{
		cache:       newMemoryCache(),
		idempotency: newMemoryIdempotencyStore(),
		rateLimiter: newMemoryRateLimiter(),
		queue:       newMemoryQueue(200),
		cleanup: func(ctx context.Context) error {
			return nil
		},
	}, nil
}

func normalizeRateLimit(rps float64, burst int) int64 {
	if burst > 0 {
		return int64(burst)
	}
	if rps <= 0 {
		return 1
	}
	return int64(math.Ceil(rps))
}
