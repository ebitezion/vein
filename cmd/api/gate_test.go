package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestGateOpenMissingEstateHeader(t *testing.T) {
	app := newTestApp()
	token, err := app.generateToken("user-1", "admin", app.config.security.tokenTTL)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := app.authenticate(app.withActiveEstate(app.requireEstateRole("admin")(http.HandlerFunc(app.openGate))))
	req := httptest.NewRequest(http.MethodPost, "/v1/gate/open", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGateOpenForbiddenForNonAdmin(t *testing.T) {
	app := newTestApp()
	app.estateRoles.SetRole("user-1", "estate-1", "resident")
	token, err := app.generateToken("user-1", "admin", app.config.security.tokenTTL)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := app.authenticate(app.withActiveEstate(app.requireEstateRole("admin")(http.HandlerFunc(app.openGate))))
	req := httptest.NewRequest(http.MethodPost, "/v1/gate/open", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Estate-ID", "estate-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestGateOpenWrongEstateForbidden(t *testing.T) {
	app := newTestApp()
	token, err := app.generateToken("user-1", "admin", app.config.security.tokenTTL)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := app.authenticate(app.withActiveEstate(app.requireEstateRole("admin")(http.HandlerFunc(app.openGate))))
	req := httptest.NewRequest(http.MethodPost, "/v1/gate/open", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Estate-ID", "estate-2")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestGateOpenMissingTokenUnauthorized(t *testing.T) {
	app := newTestApp()
	handler := app.authenticate(app.withActiveEstate(app.requireEstateRole("admin")(http.HandlerFunc(app.openGate))))
	req := httptest.NewRequest(http.MethodPost, "/v1/gate/open", nil)
	req.Header.Set("X-Estate-ID", "estate-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestGateOpenExpiredTokenUnauthorized(t *testing.T) {
	app := newTestApp()
	token, err := app.generateToken("user-1", "admin", -1*time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := app.authenticate(app.withActiveEstate(app.requireEstateRole("admin")(http.HandlerFunc(app.openGate))))
	req := httptest.NewRequest(http.MethodPost, "/v1/gate/open", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Estate-ID", "estate-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestEstateRoleStoreConcurrentAccess(t *testing.T) {
	store := newMemoryEstateRoleStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			store.SetRole("user-1", "estate-1", "admin")
		}()
		go func() {
			defer wg.Done()
			_, _ = store.GetRole("user-1", "estate-1")
		}()
	}

	wg.Wait()
}
