package main

import (
	"context"
	"sync"
)

type Job struct {
	Name    string            `json:"name"`
	Payload map[string]string `json:"payload"`
}

type JobHandler func(context.Context, Job)

type memoryQueue struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
	jobs     chan Job
}

func newMemoryQueue(size int) *memoryQueue {
	if size < 1 {
		size = 100
	}

	return &memoryQueue{
		handlers: make(map[string]JobHandler),
		jobs:     make(chan Job, size),
	}
}

func (q *memoryQueue) Register(name string, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[name] = handler
}

func (q *memoryQueue) Publish(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.jobs <- job:
		return nil
	}
}

func (q *memoryQueue) Start(ctx context.Context, workers int) {
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-q.jobs:
					q.handle(ctx, job)
				}
			}
		}()
	}
}

func (q *memoryQueue) handle(ctx context.Context, job Job) {
	q.mu.RLock()
	handler, found := q.handlers[job.Name]
	q.mu.RUnlock()
	if !found {
		return
	}

	handler(ctx, job)
}
