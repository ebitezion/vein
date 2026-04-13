package main

import (
	"context"
	"sync"
)

type Hook func(context.Context) error

type lifecycle struct {
	mu         sync.RWMutex
	onStart    []Hook
	onShutdown []Hook
}

func newLifecycle() *lifecycle {
	return &lifecycle{}
}

func (lc *lifecycle) RegisterOnStart(hook Hook) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.onStart = append(lc.onStart, hook)
}

func (lc *lifecycle) RegisterOnShutdown(hook Hook) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.onShutdown = append(lc.onShutdown, hook)
}

func (lc *lifecycle) Start(ctx context.Context) error {
	lc.mu.RLock()
	hooks := append([]Hook{}, lc.onStart...)
	lc.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (lc *lifecycle) Shutdown(ctx context.Context) error {
	lc.mu.RLock()
	hooks := append([]Hook{}, lc.onShutdown...)
	lc.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}
