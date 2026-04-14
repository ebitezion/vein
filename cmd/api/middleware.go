package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	userRoleContextKey  contextKey = "user_role"
)

func (app *application) chain(next http.Handler) http.Handler {
	middlewares := []func(http.Handler) http.Handler{
		app.recoverPanic,
		app.metricsMiddleware,
		app.tracingMiddleware,
		app.requestID,
		app.requestLogger,
		app.securityHeaders,
		app.cors,
		app.rateLimit,
		app.idempotency,
	}

	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}

	return next
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				app.serverErrorResponse(w, r)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if strings.TrimSpace(requestID) == "" {
			requestID = generateID(16)
		}

		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		payload := map[string]interface{}{
			"ts":          time.Now().UTC().Format(time.RFC3339),
			"method":      r.Method,
			"path":        r.URL.Path,
			"status_code": wrapped.statusCode,
			"duration_ms": time.Since(startedAt).Milliseconds(),
			"request_id":  app.requestIDFromContext(r.Context()),
		}

		app.logJSON(payload)
	})
}

func (app *application) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

func (app *application) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if app.isTrustedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) isTrustedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	for _, trusted := range app.config.security.corsTrustedOrigins {
		if trusted == "*" || trusted == origin {
			return true
		}
	}

	return false
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := app.readClientIP(r)
		limiterKey := "general:" + ip

		rps := app.config.security.rateLimitRPS
		burst := app.config.security.rateLimitBurst
		if r.URL.Path == "/v1/auth/token" {
			rps = app.config.security.authRateLimitRPS
			burst = app.config.security.authRateLimitBurst
			limiterKey = "auth:" + ip
		}

		allowed, err := app.rateLimiter.Allow(r.Context(), limiterKey, rps, burst)
		if err != nil {
			app.serverErrorResponse(w, r)
			return
		}
		if !allowed {
			app.rateLimitExceededResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPatch && r.Method != http.MethodPut {
			next.ServeHTTP(w, r)
			return
		}

		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		cacheKey := "idem:" + r.Method + ":" + r.URL.Path + ":" + key
		reserved, err := app.idemStore.Reserve(r.Context(), cacheKey, 24*time.Hour)
		if err != nil {
			app.serverErrorResponse(w, r)
			return
		}
		if !reserved {
			app.errorResponse(w, r, http.StatusConflict, "Duplicate request detected")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func generateID(size int) string {
	buffer := make([]byte, size)
	_, err := rand.Read(buffer)
	if err != nil {
		return time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(buffer)
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (srw *statusResponseWriter) WriteHeader(code int) {
	srw.statusCode = code
	srw.ResponseWriter.WriteHeader(code)
}

func (app *application) requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok {
		return ""
	}
	return requestID
}

func (app *application) readClientIP(r *http.Request) string {
	remoteIP := parseRemoteIP(r.RemoteAddr)
	if remoteIP == "" {
		return r.RemoteAddr
	}

	if !app.isFromTrustedProxy(remoteIP) {
		return remoteIP
	}

	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			candidate := strings.TrimSpace(parts[0])
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if net.ParseIP(realIP) != nil {
		return realIP
	}

	return remoteIP
}

func parseRemoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return strings.TrimSpace(host)
}

func (app *application) isFromTrustedProxy(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	for _, network := range app.config.security.trustedProxies {
		if network.Contains(parsed) {
			return true
		}
	}

	return false
}
