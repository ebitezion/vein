package main

import (
	"context"
	"sync"
)

type Plugin interface {
	Name() string
	Register(*application)
}

type pluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

func newPluginRegistry() *pluginRegistry {
	return &pluginRegistry{plugins: make(map[string]Plugin)}
}

func (pr *pluginRegistry) Register(plugin Plugin, app *application) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.plugins[plugin.Name()] = plugin
	plugin.Register(app)
}

func (pr *pluginRegistry) Names() []string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	names := make([]string, 0, len(pr.plugins))
	for name := range pr.plugins {
		names = append(names, name)
	}
	return names
}

type eventBus struct {
	mu       sync.RWMutex
	handlers map[string][]func(context.Context, map[string]string)
}

func newEventBus() *eventBus {
	return &eventBus{handlers: make(map[string][]func(context.Context, map[string]string))}
}

func (eb *eventBus) Subscribe(event string, handler func(context.Context, map[string]string)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[event] = append(eb.handlers[event], handler)
}

func (eb *eventBus) Publish(ctx context.Context, event string, payload map[string]string) {
	eb.mu.RLock()
	handlers := append([]func(context.Context, map[string]string){}, eb.handlers[event]...)
	eb.mu.RUnlock()

	for _, handler := range handlers {
		handler(ctx, payload)
	}
}
