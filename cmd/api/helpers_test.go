package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper to build a minimal application instance for handler/helper tests.
func newTestApp() *application {
	return &application{
		config: config{appName: "vein", version: "1.0.0", env: "test"},
		log:    logDiscard,
	}
}

// discard logger reused across tests.
var logDiscard = log.New(io.Discard, "", 0)

func TestWriteJSON(t *testing.T) {
	app := newTestApp()

	rr := httptest.NewRecorder()
	err := app.writeJSON(rr, http.StatusAccepted, envelope{"message": "ok"}, nil)

	if err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status code: want %d got %d", http.StatusAccepted, rr.Code)
	}

	got := rr.Body.String()
	want := "{\"message\":\"ok\"}\n"
	if got != want {
		t.Fatalf("body mismatch: want %q got %q", want, got)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type: want application/json got %s", ct)
	}
}

func TestReadJSON(t *testing.T) {
	app := newTestApp()

	payload := `{"name":"vein"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	var dst struct {
		Name string `json:"name"`
	}

	if err := app.readJSON(rr, req, &dst); err != nil {
		t.Fatalf("readJSON returned error: %v", err)
	}

	if dst.Name != "vein" {
		t.Fatalf("decoded name: want vein got %s", dst.Name)
	}
}

func TestReadJSONUnknownField(t *testing.T) {
	app := newTestApp()

	payload := `{"name":"vein","age":10}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	var dst struct {
		Name string `json:"name"`
	}

	err := app.readJSON(rr, req, &dst)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}

	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}
