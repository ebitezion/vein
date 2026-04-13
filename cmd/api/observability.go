package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type metricsStore struct {
	totalRequests uint64
	status2xx     uint64
	status4xx     uint64
	status5xx     uint64
	latencyTotal  int64
	mu            sync.Mutex
	byPath        map[string]uint64
}

func newMetricsStore() *metricsStore {
	return &metricsStore{byPath: make(map[string]uint64)}
}

func (m *metricsStore) record(path string, status int, duration time.Duration) {
	atomic.AddUint64(&m.totalRequests, 1)
	atomic.AddInt64(&m.latencyTotal, duration.Milliseconds())

	switch {
	case status >= 200 && status < 300:
		atomic.AddUint64(&m.status2xx, 1)
	case status >= 400 && status < 500:
		atomic.AddUint64(&m.status4xx, 1)
	case status >= 500:
		atomic.AddUint64(&m.status5xx, 1)
	}

	m.mu.Lock()
	m.byPath[path]++
	m.mu.Unlock()
}

func (m *metricsStore) snapshot() envelope {
	total := atomic.LoadUint64(&m.totalRequests)
	latency := atomic.LoadInt64(&m.latencyTotal)
	average := float64(0)
	if total > 0 {
		average = float64(latency) / float64(total)
	}

	m.mu.Lock()
	pathTotals := make(map[string]uint64, len(m.byPath))
	for k, v := range m.byPath {
		pathTotals[k] = v
	}
	m.mu.Unlock()

	return envelope{
		"total_requests":       total,
		"total_2xx":            atomic.LoadUint64(&m.status2xx),
		"total_4xx":            atomic.LoadUint64(&m.status4xx),
		"total_5xx":            atomic.LoadUint64(&m.status5xx),
		"avg_latency_ms":       average,
		"requests_by_endpoint": pathTotals,
	}
}

func (app *application) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		app.metrics.record(r.URL.Path, wrapped.statusCode, time.Since(started))
	})
}

func (app *application) metricsHandler(w http.ResponseWriter, r *http.Request) {
	data := envelope{"metrics": app.metrics.snapshot()}
	_ = app.writeJSON(w, http.StatusOK, data, nil)
}

func (app *application) logJSON(payload map[string]interface{}) {
	if payload == nil {
		return
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		app.log.Printf("[json-log] marshal error: %v", err)
		return
	}

	app.log.Println(string(encoded))
}
