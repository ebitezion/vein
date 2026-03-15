package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	app := newTestApp()

	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	app.healthcheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code: want %d got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "\"AppName\":\"vein\"") {
		t.Fatalf("expected AppName in response, got %s", body)
	}
	if !strings.Contains(body, "\"Version\":\"1.0.0\"") {
		t.Fatalf("expected Version in response, got %s", body)
	}
	if !strings.Contains(body, "\"MY_ENV\":\"test\"") {
		t.Fatalf("expected MY_ENV in response, got %s", body)
	}
}
