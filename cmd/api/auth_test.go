package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	app := newTestApp()
	app.config.security.tokenSecret = "test-secret"

	token, err := app.generateToken("user-1", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}
	claims, err := app.parseToken(token)
	if err != nil {
		t.Fatalf("parseToken returned error: %v", err)
	}

	if claims.Role != "admin" {
		t.Fatalf("expected role admin, got %s", claims.Role)
	}
}

func TestAuthenticateMiddleware(t *testing.T) {
	app := newTestApp()
	app.config.security.tokenSecret = "test-secret"

	token, err := app.generateToken("user-1", "manager", time.Hour)
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}
	handler := app.authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
