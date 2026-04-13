package main

import (
	"context"
	"sync"
	"time"
)

type memoryCacheItem struct {
	value     string
	expiresAt time.Time
}

type memoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryCacheItem
}

func newMemoryCache() *memoryCache {
	return &memoryCache{items: make(map[string]memoryCacheItem)}
}

func (c *memoryCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = memoryCacheItem{value: value, expiresAt: expiresAt}
	return nil
}

func (c *memoryCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()
	if !found {
		return "", errCacheMiss
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return "", errCacheMiss
	}

	return item.value, nil
}

type memoryIdempotencyStore struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func newMemoryIdempotencyStore() *memoryIdempotencyStore {
	return &memoryIdempotencyStore{items: make(map[string]time.Time)}
}

func (m *memoryIdempotencyStore) Reserve(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for k, expiry := range m.items {
		if now.After(expiry) {
			delete(m.items, k)
		}
	}

	if expiry, exists := m.items[key]; exists && now.Before(expiry) {
		return false, nil
	}

	m.items[key] = now.Add(ttl)
	return true, nil
}
