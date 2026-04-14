package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

func newRedisCache(client *redis.Client) *redisCache {
	return &redisCache{client: client}
}

func (r *redisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, "cache:"+key, value, ttl).Err()
}

func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, "cache:"+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", errCacheMiss
	}
	return value, err
}

type redisIdempotencyStore struct {
	client *redis.Client
}

func newRedisIdempotencyStore(client *redis.Client) *redisIdempotencyStore {
	return &redisIdempotencyStore{client: client}
}

func (r *redisIdempotencyStore) Reserve(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	result, err := r.client.SetArgs(ctx, "idempotency:"+key, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()
	return result == "OK", err
}

type redisRateLimiter struct {
	client *redis.Client
}

func newRedisRateLimiter(client *redis.Client) *redisRateLimiter {
	return &redisRateLimiter{client: client}
}

func (r *redisRateLimiter) Allow(ctx context.Context, key string, rps float64, burst int) (bool, error) {
	limit := normalizeRateLimit(rps, burst)
	windowKey := fmt.Sprintf("rate:%s:%d", key, time.Now().UTC().Unix())
	count, err := r.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		if err := r.client.Expire(ctx, windowKey, 2*time.Second).Err(); err != nil {
			return false, err
		}
	}

	return count <= limit, nil
}

type redisQueue struct {
	client   *redis.Client
	queueKey string
	handlers map[string]JobHandler
}

func newRedisQueue(client *redis.Client, queueKey string) *redisQueue {
	return &redisQueue{
		client:   client,
		queueKey: queueKey,
		handlers: make(map[string]JobHandler),
	}
}

func (q *redisQueue) Register(name string, handler JobHandler) {
	q.handlers[name] = handler
}

func (q *redisQueue) Publish(ctx context.Context, job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.queueKey, payload).Err()
}

func (q *redisQueue) Start(ctx context.Context, workers int) {
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		go q.worker(ctx)
	}
}

func (q *redisQueue) worker(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		result, err := q.client.BRPop(ctx, 2*time.Second, q.queueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			continue
		}
		if len(result) != 2 {
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			continue
		}

		handler, ok := q.handlers[job.Name]
		if !ok {
			continue
		}

		handler(ctx, job)
	}
}
