package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// config type allows for system configuration
type config struct {
	port    int
	env     string
	appName string
	version string
	db      struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	security struct {
		corsTrustedOrigins []string
		rateLimitRPS       float64
		rateLimitBurst     int
		authRateLimitRPS   float64
		authRateLimitBurst int
		tokenSecret        string
		tokenIssuer        string
		tokenAudience      string
		tokenTTL           time.Duration
		trustedProxies     []*net.IPNet
	}
	redis struct {
		addr     string
		password string
		db       int
		queueKey string
		enabled  bool
	}
	tracing struct {
		enabled      bool
		otlpEndpoint string
		sampleRatio  float64
	}
}

func loadConfig() (config, error) {
	cfg := config{}

	cfg.appName = getEnv("APP_NAME", "vein")
	cfg.version = getEnv("APP_VERSION", "1.0.0")
	cfg.env = getEnv("MY_ENV", "development")

	port, err := strconv.Atoi(getEnv("PORT", "4000"))
	if err != nil {
		return cfg, fmt.Errorf("invalid PORT value: %w", err)
	}
	cfg.port = port

	cfg.db.dsn = strings.TrimSpace(getSecretEnv("DB_DSN", ""))
	cfg.db.maxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	cfg.db.maxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", 25)
	cfg.db.maxIdleTime = getEnv("DB_MAX_IDLE_TIME", "15m")

	cfg.security.corsTrustedOrigins = parseCSV(getEnv("CORS_TRUSTED_ORIGINS", "http://localhost:3000,http://localhost:5173"))
	cfg.security.rateLimitRPS = getEnvFloat("RATE_LIMIT_RPS", 5)
	cfg.security.rateLimitBurst = getEnvInt("RATE_LIMIT_BURST", 10)
	cfg.security.authRateLimitRPS = getEnvFloat("AUTH_RATE_LIMIT_RPS", 1)
	cfg.security.authRateLimitBurst = getEnvInt("AUTH_RATE_LIMIT_BURST", 3)
	cfg.security.tokenSecret = getSecretEnv("TOKEN_SECRET", "replace-me-in-production")
	cfg.security.tokenIssuer = getEnv("TOKEN_ISSUER", cfg.appName)
	cfg.security.tokenAudience = getEnv("TOKEN_AUDIENCE", "vein-clients")
	cfg.security.tokenTTL = getEnvDuration("TOKEN_TTL", 24*time.Hour)
	cfg.security.trustedProxies = parseTrustedProxies(getEnv("TRUSTED_PROXIES", ""))

	cfg.redis.addr = getEnv("REDIS_ADDR", "")
	cfg.redis.password = getSecretEnv("REDIS_PASSWORD", "")
	cfg.redis.db = getEnvInt("REDIS_DB", 0)
	cfg.redis.queueKey = getEnv("REDIS_QUEUE_KEY", "vein:jobs")
	cfg.redis.enabled = cfg.redis.addr != ""

	cfg.tracing.enabled = getEnvBool("OTEL_ENABLED", false)
	cfg.tracing.otlpEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	cfg.tracing.sampleRatio = getEnvFloat("OTEL_SAMPLE_RATIO", 1.0)

	if err := validateConfig(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func validateConfig(cfg config) error {
	var errs []string

	if cfg.port < 1 || cfg.port > 65535 {
		errs = append(errs, "PORT must be between 1 and 65535")
	}

	if cfg.appName == "" {
		errs = append(errs, "APP_NAME must be provided")
	}

	if cfg.env != "development" && cfg.env != "staging" && cfg.env != "production" && cfg.env != "test" {
		errs = append(errs, "MY_ENV must be one of development, staging, production, test")
	}

	if cfg.db.maxOpenConns < 1 {
		errs = append(errs, "DB_MAX_OPEN_CONNS must be greater than 0")
	}

	if cfg.db.maxIdleConns < 0 {
		errs = append(errs, "DB_MAX_IDLE_CONNS must be greater than or equal to 0")
	}

	if _, err := time.ParseDuration(cfg.db.maxIdleTime); err != nil {
		errs = append(errs, "DB_MAX_IDLE_TIME must be a valid duration")
	}

	if cfg.security.rateLimitRPS <= 0 {
		errs = append(errs, "RATE_LIMIT_RPS must be greater than 0")
	}

	if cfg.security.rateLimitBurst < 1 {
		errs = append(errs, "RATE_LIMIT_BURST must be greater than 0")
	}
	if cfg.security.authRateLimitRPS <= 0 {
		errs = append(errs, "AUTH_RATE_LIMIT_RPS must be greater than 0")
	}
	if cfg.security.authRateLimitBurst < 1 {
		errs = append(errs, "AUTH_RATE_LIMIT_BURST must be greater than 0")
	}

	if cfg.security.tokenSecret == "" {
		errs = append(errs, "TOKEN_SECRET must be provided")
	}
	if cfg.env != "test" && cfg.security.tokenSecret == "replace-me-in-production" {
		errs = append(errs, "TOKEN_SECRET must be changed")
	}
	if cfg.env != "test" && !isStrongSecret(cfg.security.tokenSecret) {
		errs = append(errs, "TOKEN_SECRET is too weak; use at least 32 chars with mixed character classes")
	}
	if cfg.security.tokenTTL < 5*time.Minute {
		errs = append(errs, "TOKEN_TTL must be at least 5m")
	}
	if cfg.security.tokenTTL > 24*time.Hour {
		errs = append(errs, "TOKEN_TTL must be at most 24h")
	}
	if cfg.tracing.sampleRatio < 0 || cfg.tracing.sampleRatio > 1 {
		errs = append(errs, "OTEL_SAMPLE_RATIO must be between 0 and 1")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseCSV(raw string) []string {
	items := strings.Split(raw, ",")
	parsed := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			parsed = append(parsed, trimmed)
		}
	}
	return parsed
}

func getSecretEnv(key, fallback string) string {
	filePath := strings.TrimSpace(os.Getenv(key + "_FILE"))
	if filePath != "" {
		value, err := readSecretFile(filePath)
		if err == nil {
			secret := strings.TrimSpace(string(value))
			if secret != "" {
				return secret
			}
		}
	}

	return getEnv(key, fallback)
}

func parseTrustedProxies(raw string) []*net.IPNet {
	items := parseCSV(raw)
	trusted := make([]*net.IPNet, 0, len(items))
	for _, item := range items {
		if ip := net.ParseIP(item); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			trusted = append(trusted, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
			continue
		}

		_, network, err := net.ParseCIDR(item)
		if err == nil {
			trusted = append(trusted, network)
		}
	}
	return trusted
}

func isStrongSecret(secret string) bool {
	if len(secret) < 32 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range secret {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	classes := 0
	if hasUpper {
		classes++
	}
	if hasLower {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSymbol {
		classes++
	}

	return classes >= 3
}

func readSecretFile(path string) ([]byte, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" {
		return nil, fmt.Errorf("empty secret file path")
	}

	dir := filepath.Dir(clean)
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) {
		return nil, fmt.Errorf("invalid secret file path")
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	file, err := root.Open(base)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}
