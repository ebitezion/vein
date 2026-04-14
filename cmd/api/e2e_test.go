package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ebitezion/vein/internal/data"
	_ "github.com/lib/pq"
)

func TestE2EUsersListAndMiddleware(t *testing.T) {
	dsn := os.Getenv("INTEGRATION_TEST_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("INTEGRATION_TEST_DSN not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := applySQLFile(db, filepath.Join("..", "..", "migrations", "000001_create_users.down.sql")); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
	if err := applySQLFile(db, filepath.Join("..", "..", "migrations", "000001_create_users.up.sql")); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}
	if err := applySQLFile(db, filepath.Join("..", "..", "seeds", "000001_seed_users.sql")); err != nil {
		t.Fatalf("apply seed: %v", err)
	}

	cfg := config{port: 4000, env: "test", appName: "vein", version: "1.0.0"}
	cfg.security.tokenSecret = "integration-secret"
	cfg.security.tokenIssuer = "vein"
	cfg.security.tokenAudience = "vein-clients"
	cfg.security.tokenTTL = time.Hour
	cfg.security.corsTrustedOrigins = []string{"http://localhost:3000"}
	cfg.security.rateLimitRPS = 100
	cfg.security.rateLimitBurst = 100

	app, err := newApplication(cfg, log.New(io.Discard, "", 0), data.NewModels(db))
	if err != nil {
		t.Fatalf("new application: %v", err)
	}
	app.registerDefaults()

	server := httptest.NewServer(app.routes())
	defer server.Close()

	token := issueAuthToken(t, server.URL)

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/users?page=1&page_size=10&sort=created_at", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("users request failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d body=%s", res.StatusCode, string(body))
	}

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "admin@vein.dev") {
		t.Fatalf("expected seeded user in response, got %s", string(body))
	}

	idempotencyReqBody := []byte(`{"action":"test"}`)
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/jobs/audit", bytes.NewReader(idempotencyReqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "idem-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("idempotency request failed: %v", err)
		}
		resp.Body.Close()

		if i == 0 && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("first idempotent request expected 202, got %d", resp.StatusCode)
		}
		if i == 1 && resp.StatusCode != http.StatusConflict {
			t.Fatalf("second idempotent request expected 409, got %d", resp.StatusCode)
		}
	}
}

func issueAuthToken(t *testing.T, serverURL string) string {
	t.Helper()

	payload := map[string]string{"email": "admin@vein.dev", "password": "VeinPass#2026!"}
	raw, _ := json.Marshal(payload)
	res, err := http.Post(serverURL+"/v1/auth/token", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 201 token response, got %d body=%s", res.StatusCode, string(body))
	}

	var envelope map[string]map[string]string
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode token response: %v", err)
	}

	token := envelope["auth"]["token"]
	if token == "" {
		t.Fatal("expected auth token")
	}
	return token
}

func applySQLFile(db *sql.DB, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}
