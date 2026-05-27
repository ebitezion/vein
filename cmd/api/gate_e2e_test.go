package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ebitezion/vein/internal/data"
)

func TestE2EGateOpenPermissionDrift(t *testing.T) {
	cfg := config{port: 4000, env: "test", appName: "vein", version: "1.0.0"}
	cfg.security.tokenSecret = "integration-secret"
	cfg.security.tokenIssuer = "vein"
	cfg.security.tokenAudience = "vein-clients"
	cfg.security.tokenTTL = time.Hour
	cfg.security.rateLimitRPS = 100
	cfg.security.rateLimitBurst = 100
	cfg.security.authRateLimitRPS = 100
	cfg.security.authRateLimitBurst = 100

	app, err := newApplication(cfg, log.New(io.Discard, "", 0), data.Models{})
	if err != nil {
		t.Fatalf("new application: %v", err)
	}
	app.registerDefaults()

	server := httptest.NewServer(app.routes())
	defer server.Close()

	token, err := app.generateToken("user-1", "admin", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	callGate := func() int {
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/gate/open", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Estate-ID", "estate-1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("gate request failed: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if code := callGate(); code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", code)
	}

	app.estateRoles.SetRole("user-1", "estate-1", "resident")

	if code := callGate(); code != http.StatusForbidden {
		t.Fatalf("expected second request 403 after downgrade, got %d", code)
	}
}
