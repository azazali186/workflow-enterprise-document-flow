// Package config loads and validates environment configuration.
package config

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"time"
)

// Config aggregates every tunable in the backend.
type Config struct {
	Port              string
	Env               string
	LogLevel          string
	DatabaseURL       string
	RedisURL          string
	NATSURL           string
	JWTSecret         string
	JWTExpiry         time.Duration
	RefreshExpiry     time.Duration
	EncryptionKey     string
	RateLimit         int
	MaxBodyBytes      int
	CORSOrigins       []string
	TrustedProxies    []string
	AdminEmail        string
	AdminPassword     string
	AdminName         string
	OutboxInterval    time.Duration
	OutboxBatch       int
	OutboxMaxAttempts int
	SwaggerEnabled    bool
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	StorageDir        string
	S3Endpoint        string
	S3AccessKey       string
	S3SecretKey       string
	S3Bucket          string
	S3Region          string
	S3UseSSL          bool
	VirusScanner      string
	ClamAVAddr        string
	ClamAVTimeout     time.Duration
	Indexer           string
	OpenSearchURL     string
	OpenSearchIndex   string
	OpenSearchUser    string
	OpenSearchPass    string
}

// Known-weak values that must never be accepted in production. These are the
// defaults shipped in docker-compose / .env.example so an operator who forgets
// to override them gets a hard startup failure instead of a breach.
var (
	weakJWTSecrets = []string{
		"change-me-in-prod-please-32bytes",
		"change-me-to-a-long-random-secret",
		"changeme",
		"secret",
		// The development fallback baked into Load (see below): production must
		// never sign tokens with a key that ships in source code.
		"yoYL2iDWHegx30LfD9Vz3t0LKcpHZ7H77x1ImRiIfbIzc+6zcCVSrm1YgIwZhdTE",
	}
	weakEncryptionKeys = []string{
		"/7DNZtaYwk9xCd7gLiO3fjBLz8Sm3WAOQiE2QZ53Vt8=",
		// The deterministic development fallback (see Load): production must
		// never run on a key that ships in source code.
		"gLKrBZXjEQOnP33JgHQ5p3tX+KXzprNhZVa6+7il6SY=",
	}
	weakAdminPasswords = []string{
		"ChangeMe123!",
	}
)

// Load reads configuration from the environment.
func Load() (*Config, error) {
	c := &Config{
		Port:            getEnv("PORT", "8090"),
		Env:             getEnv("ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/docu_flow?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		NATSURL:         getEnv("NATS_URL", "nats://localhost:4222"),
		JWTSecret:       getEnv("JWT_SECRET", "yoYL2iDWHegx30LfD9Vz3t0LKcpHZ7H77x1ImRiIfbIzc+6zcCVSrm1YgIwZhdTE"),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", "gLKrBZXjEQOnP33JgHQ5p3tX+KXzprNhZVa6+7il6SY="),
		AdminEmail:      getEnv("ADMIN_EMAIL", "admin@aeroxe.io"),
		AdminPassword:   getEnv("ADMIN_PASSWORD", "ChangeMe123!"),
		AdminName:       getEnv("ADMIN_NAME", "Super Admin"),
		StorageDir:      getEnv("STORAGE_DIR", "./storage"),
		S3Endpoint:      getEnv("S3_ENDPOINT", ""),
		S3AccessKey:     getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:     getEnv("S3_SECRET_KEY", ""),
		S3Bucket:        getEnv("S3_BUCKET", "docuflow"),
		S3Region:        getEnv("S3_REGION", "us-east-1"),
		S3UseSSL:        getEnv("S3_USE_SSL", "true") == "true",
		CORSOrigins:     splitCSV(getEnv("CORS_ORIGINS", "*")),
		TrustedProxies:  splitCSV(getEnv("TRUSTED_PROXIES", "")),
		VirusScanner:    getEnv("VIRUS_SCANNER", "none"),
		ClamAVAddr:      getEnv("CLAMAV_ADDR", "localhost:3310"),
		Indexer:         getEnv("INDEXER", "none"),
		OpenSearchURL:   getEnv("OPENSEARCH_URL", ""),
		OpenSearchIndex: getEnv("OPENSEARCH_INDEX", "documents"),
		OpenSearchUser:  getEnv("OPENSEARCH_USERNAME", ""),
		OpenSearchPass:  getEnv("OPENSEARCH_PASSWORD", ""),
	}
	var err error
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	// An empty ENCRYPTION_KEY falls back to a deterministic development key
	// which must never be used outside local development.
	if c.EncryptionKey == "" && c.IsProduction() {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required in production (32 bytes, base64)")
	}
	if c.JWTExpiry, err = time.ParseDuration(getEnv("JWT_EXPIRY", "24h")); err != nil {
		return nil, fmt.Errorf("JWT_EXPIRY invalid: %w", err)
	}
	if c.RefreshExpiry, err = time.ParseDuration(getEnv("REFRESH_EXPIRY", "168h")); err != nil {
		return nil, fmt.Errorf("REFRESH_EXPIRY invalid: %w", err)
	}
	if c.ClamAVTimeout, err = time.ParseDuration(getEnv("CLAMAV_TIMEOUT", "30s")); err != nil {
		return nil, fmt.Errorf("CLAMAV_TIMEOUT invalid: %w", err)
	}
	if c.OutboxInterval, err = time.ParseDuration(getEnv("OUTBOX_INTERVAL", "2s")); err != nil {
		return nil, fmt.Errorf("OUTBOX_INTERVAL invalid: %w", err)
	}
	c.RateLimit = getEnvInt("RATE_LIMIT", 120)
	c.MaxBodyBytes = getEnvInt("MAX_BODY_SIZE_MB", 10) * 1024 * 1024
	c.OutboxBatch = getEnvInt("OUTBOX_BATCH_SIZE", 50)
	c.OutboxMaxAttempts = getEnvInt("OUTBOX_MAX_ATTEMPTS", 5)
	c.DBMaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	c.DBMaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", 10)
	// Swagger UI is on by default in development and off in production unless
	// explicitly enabled (SWAGGER_ENABLED=true). Exposing the full API schema
	// in production is an information-disclosure risk operators must opt into.
	switch swagger := getEnv("SWAGGER_ENABLED", ""); swagger {
	case "true":
		c.SwaggerEnabled = true
	case "false":
		c.SwaggerEnabled = false
	default:
		c.SwaggerEnabled = !c.IsProduction()
	}

	if err := c.validatePipeline(); err != nil {
		return nil, err
	}
	if c.IsProduction() {
		if err := c.validateProduction(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// validateProduction rejects weak or absent security settings when ENV is
// "production". Development keeps permissive defaults for local ergonomics.
func (c *Config) validateProduction() error {
	if slices.Contains(weakJWTSecrets, c.JWTSecret) {
		return fmt.Errorf("JWT_SECRET is a known dev default; set a strong random secret in production")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if slices.Contains(weakEncryptionKeys, c.EncryptionKey) {
		return fmt.Errorf("ENCRYPTION_KEY is the known dev default; set a fresh 32-byte base64 key in production")
	}
	if slices.Contains(weakAdminPasswords, c.AdminPassword) {
		return fmt.Errorf("ADMIN_PASSWORD is the known dev default; set a strong password in production")
	}
	if len(c.AdminPassword) < 12 {
		return fmt.Errorf("ADMIN_PASSWORD must be at least 12 characters in production")
	}
	if len(c.CORSOrigins) == 0 || slices.Contains(c.CORSOrigins, "*") {
		return fmt.Errorf("CORS_ORIGINS must be an explicit allow-list (no '*') in production")
	}
	return nil
}

// validatePipeline rejects unknown scanner/indexer settings and requires the
// endpoint for a configured integration. Runs in every environment so a typo
// fails fast instead of silently disabling security features.
func (c *Config) validatePipeline() error {
	if c.VirusScanner != "none" && c.VirusScanner != "clamav" {
		return fmt.Errorf("VIRUS_SCANNER must be 'none' or 'clamav', got %q", c.VirusScanner)
	}
	if c.VirusScanner == "clamav" && c.ClamAVAddr == "" {
		return fmt.Errorf("CLAMAV_ADDR is required when VIRUS_SCANNER=clamav")
	}
	if c.Indexer != "none" && c.Indexer != "opensearch" {
		return fmt.Errorf("INDEXER must be 'none' or 'opensearch', got %q", c.Indexer)
	}
	if c.Indexer == "opensearch" && c.OpenSearchURL == "" {
		return fmt.Errorf("OPENSEARCH_URL is required when INDEXER=opensearch")
	}
	return nil
}

// LoadMigrations reads the minimal configuration schema migrations need,
// bypassing the web-app production validation so ops can run
// `cmd/migrate up` with only DATABASE_URL set. The admin bootstrap fields are
// still read so the seeded admin matches the one the server would create.
func LoadMigrations() (*Config, error) {
	c := &Config{
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@aeroxe.io"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "ChangeMe123!"),
		AdminName:     getEnv("ADMIN_NAME", "Super Admin"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

// IsProduction reports whether the process runs in production mode.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// TrustedProxyNets parses TRUSTED_PROXIES into CIDR ranges. When the list is
// empty no proxy header is trusted and the direct peer address is used, which
// prevents X-Forwarded-For spoofing of the rate-limit key.
func (c *Config) TrustedProxyNets() ([]*net.IPNet, error) {
	out := make([]*net.IPNet, 0, len(c.TrustedProxies))
	for _, s := range c.TrustedProxies {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES entry %q is not a valid CIDR: %w", s, err)
		}
		out = append(out, n)
	}
	return out, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range splitOnComma(s) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitOnComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}
