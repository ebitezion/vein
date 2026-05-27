package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ebitezion/vein/internal/data"
	"golang.org/x/crypto/bcrypt"
)

func TestIssueTokenUnauthorizedWithInvalidPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass#2026"), bcrypt.DefaultCost)
	rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "email", "phone", "password_hash", "role", "status", "email_verified", "created_at", "updated_at"}).
		AddRow("u1", "Admin", "User", "admin@vein.dev", "+10000000001", string(hash), "admin", "active", true, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, first_name, last_name, email, phone, password_hash, role, status, email_verified, created_at, updated_at\s+FROM users\s+WHERE lower\(email\) = \$1`).
		WithArgs("admin@vein.dev").
		WillReturnRows(rows)

	app := newTestApp()
	app.model = data.NewModels(db)

	payload, _ := json.Marshal(map[string]string{"email": "admin@vein.dev", "password": "wrong-password"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.issueToken(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestIssueTokenForbiddenForDisabledUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	password := "CorrectPass#2026"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "email", "phone", "password_hash", "role", "status", "email_verified", "created_at", "updated_at"}).
		AddRow("u2", "Disabled", "User", "disabled@vein.dev", "+10000000004", string(hash), "manager", "disabled", true, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, first_name, last_name, email, phone, password_hash, role, status, email_verified, created_at, updated_at\s+FROM users\s+WHERE lower\(email\) = \$1`).
		WithArgs("disabled@vein.dev").
		WillReturnRows(rows)

	app := newTestApp()
	app.model = data.NewModels(db)

	payload, _ := json.Marshal(map[string]string{"email": "disabled@vein.dev", "password": password})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.issueToken(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestReadClientIPIgnoresSpoofedForwardedHeadersWithoutTrustedProxy(t *testing.T) {
	app := newTestApp()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.15:4444"
	req.Header.Set("X-Forwarded-For", "198.51.100.2")
	req.Header.Set("X-Real-IP", "198.51.100.3")

	got := app.readClientIP(req)
	if got != "203.0.113.15" {
		t.Fatalf("expected remote ip fallback, got %s", got)
	}
}

func TestReadClientIPUsesForwardedHeadersForTrustedProxy(t *testing.T) {
	app := newTestApp()
	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	app.config.security.trustedProxies = []*net.IPNet{trustedCIDR}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.2, 10.0.0.5")

	got := app.readClientIP(req)
	if got != "198.51.100.2" {
		t.Fatalf("expected forwarded ip, got %s", got)
	}
}

func TestAuditRouteRejectsAnonymousRequests(t *testing.T) {
	app := newTestApp()
	app.config.security.rateLimitRPS = 100
	app.config.security.rateLimitBurst = 100
	app.config.security.authRateLimitRPS = 100
	app.config.security.authRateLimitBurst = 100

	handler := app.routes()
	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/audit", strings.NewReader(`{"action":"audit"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthEndpointHasStricterRateLimit(t *testing.T) {
	app := newTestApp()
	app.config.security.rateLimitRPS = 100
	app.config.security.rateLimitBurst = 100
	app.config.security.authRateLimitRPS = 1
	app.config.security.authRateLimitBurst = 1

	handler := app.routes()

	req1 := httptest.NewRequest(http.MethodPost, "/v1/auth/token", strings.NewReader(`{}`))
	req1.RemoteAddr = "203.0.113.30:1234"
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/token", strings.NewReader(`{}`))
	req2.RemoteAddr = "203.0.113.30:1234"
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request should not be rate limited")
	}
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second auth request to be rate-limited, got %d", rr2.Code)
	}
}

// compile-time guard for sql package in this file.
var _ = sql.ErrNoRows
