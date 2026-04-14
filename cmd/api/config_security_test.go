package main

import (
	"strings"
	"testing"
	"time"
)

func validSecurityConfigForTest() config {
	cfg := config{}
	cfg.port = 4000
	cfg.appName = "vein"
	cfg.env = "development"
	cfg.db.maxOpenConns = 5
	cfg.db.maxIdleConns = 5
	cfg.db.maxIdleTime = "15m"
	cfg.security.rateLimitRPS = 5
	cfg.security.rateLimitBurst = 10
	cfg.security.authRateLimitRPS = 1
	cfg.security.authRateLimitBurst = 3
	cfg.security.tokenSecret = "VeryStrongTokenSecret#2026WithLongLength!"
	cfg.security.tokenTTL = time.Hour
	cfg.tracing.sampleRatio = 1
	return cfg
}

func TestValidateConfigRejectsWeakTokenSecretOutsideTest(t *testing.T) {
	cfg := validSecurityConfigForTest()
	cfg.security.tokenSecret = "weak-secret"

	err := validateConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "too weak") {
		t.Fatalf("expected weak secret validation error, got %v", err)
	}
}

func TestValidateConfigAllowsWeakTokenSecretInTestEnv(t *testing.T) {
	cfg := validSecurityConfigForTest()
	cfg.env = "test"
	cfg.security.tokenSecret = "weak-secret"

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("expected weak secret to be allowed in test env, got %v", err)
	}
}

func TestValidateConfigRejectsShortAndLongTokenTTL(t *testing.T) {
	shortTTL := validSecurityConfigForTest()
	shortTTL.security.tokenTTL = 2 * time.Minute
	if err := validateConfig(shortTTL); err == nil || !strings.Contains(err.Error(), "at least 5m") {
		t.Fatalf("expected short token ttl error, got %v", err)
	}

	longTTL := validSecurityConfigForTest()
	longTTL.security.tokenTTL = 48 * time.Hour
	if err := validateConfig(longTTL); err == nil || !strings.Contains(err.Error(), "at most 24h") {
		t.Fatalf("expected long token ttl error, got %v", err)
	}
}
