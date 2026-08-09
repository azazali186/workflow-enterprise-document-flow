package config

import (
	"strings"
	"testing"
)

const strongSecret = "a9f2c1e8b7d64f0a9c8b7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7f6"

func productionEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db?sslmode=disable")
	t.Setenv("JWT_SECRET", strongSecret)
	t.Setenv("ENCRYPTION_KEY", "2mK0FkMujW63YEPZq0tQZ6PpR9n0gYv4t4uD1p7sF2c=")
	t.Setenv("ADMIN_PASSWORD", "S3cure!Admin#Passw0rd")
	t.Setenv("CORS_ORIGINS", "https://app.example.com")
}

func TestProductionRejectsWeakJWTSecret(t *testing.T) {
	productionEnv(t)
	t.Setenv("JWT_SECRET", "change-me-in-prod-please-32bytes")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET rejection, got %v", err)
	}
}

func TestProductionRejectsSourceDefaultJWTSecret(t *testing.T) {
	productionEnv(t)
	// The dev fallback baked into Load (used when JWT_SECRET is unset).
	t.Setenv("JWT_SECRET", "yoYL2iDWHegx30LfD9Vz3t0LKcpHZ7H77x1ImRiIfbIzc+6zcCVSrm1YgIwZhdTE")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET rejection, got %v", err)
	}
}

func TestProductionRejectsSourceDefaultEncryptionKey(t *testing.T) {
	productionEnv(t)
	// The deterministic dev key from Load's fallback.
	t.Setenv("ENCRYPTION_KEY", "gLKrBZXjEQOnP33JgHQ5p3tX+KXzprNhZVa6+7il6SY=")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
		t.Fatalf("expected ENCRYPTION_KEY rejection, got %v", err)
	}
}

func TestProductionRejectsWeakEncryptionKey(t *testing.T) {
	productionEnv(t)
	t.Setenv("ENCRYPTION_KEY", "/7DNZtaYwk9xCd7gLiO3fjBLz8Sm3WAOQiE2QZ53Vt8=")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
		t.Fatalf("expected ENCRYPTION_KEY rejection, got %v", err)
	}
}

func TestProductionRejectsDefaultAdminPassword(t *testing.T) {
	productionEnv(t)
	t.Setenv("ADMIN_PASSWORD", "ChangeMe123!")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ADMIN_PASSWORD") {
		t.Fatalf("expected ADMIN_PASSWORD rejection, got %v", err)
	}
}

func TestProductionRejectsShortJWTSecret(t *testing.T) {
	productionEnv(t)
	t.Setenv("JWT_SECRET", "short-secret")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET length rejection, got %v", err)
	}
}

func TestProductionRejectsShortAdminPassword(t *testing.T) {
	productionEnv(t)
	t.Setenv("ADMIN_PASSWORD", "short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ADMIN_PASSWORD") {
		t.Fatalf("expected ADMIN_PASSWORD length rejection, got %v", err)
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	productionEnv(t)
	t.Setenv("CORS_ORIGINS", "*")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CORS_ORIGINS") {
		t.Fatalf("expected CORS_ORIGINS rejection, got %v", err)
	}
}

func TestProductionAcceptsStrongConfig(t *testing.T) {
	productionEnv(t)
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("strong production config must load: %v", err)
	}
	nets, err := cfg.TrustedProxyNets()
	if err != nil {
		t.Fatal(err)
	}
	if len(nets) != 2 {
		t.Fatalf("expected 2 trusted proxy nets, got %d", len(nets))
	}
}

func TestProductionRejectsGarbageProxyCIDR(t *testing.T) {
	productionEnv(t)
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,not-a-cidr")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("config should load with garbage proxies (validated later): %v", err)
	}
	if _, err := cfg.TrustedProxyNets(); err == nil {
		t.Fatal("expected CIDR parse error")
	}
}

func TestDevelopmentKeepsPermissiveDefaults(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "dev-secret")
	t.Setenv("ADMIN_PASSWORD", "ChangeMe123!")
	t.Setenv("CORS_ORIGINS", "*")
	t.Setenv("ENCRYPTION_KEY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("development defaults must load: %v", err)
	}
	if cfg.IsProduction() {
		t.Fatal("expected development env")
	}
}

func TestProductionRequiresEncryptionKey(t *testing.T) {
	productionEnv(t)
	t.Setenv("ENCRYPTION_KEY", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
		t.Fatalf("expected ENCRYPTION_KEY requirement, got %v", err)
	}
}

func TestSwaggerDefaultPerEnvironment(t *testing.T) {
	// Development: on by default.
	t.Setenv("ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "dev-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SwaggerEnabled {
		t.Fatal("swagger should default to enabled in development")
	}

	// Production: off by default (schema disclosure is opt-in).
	productionEnv(t)
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SwaggerEnabled {
		t.Fatal("swagger must default to disabled in production")
	}

	// Explicit override in production.
	t.Setenv("SWAGGER_ENABLED", "true")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SwaggerEnabled {
		t.Fatal("SWAGGER_ENABLED=true must force swagger on in production")
	}
}

func TestDBPoolDefaultsAndOverride(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "dev-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBMaxOpenConns != 25 || cfg.DBMaxIdleConns != 10 {
		t.Fatalf("pool defaults = %d/%d, want 25/10", cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	}
	t.Setenv("DB_MAX_OPEN_CONNS", "50")
	t.Setenv("DB_MAX_IDLE_CONNS", "5")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBMaxOpenConns != 50 || cfg.DBMaxIdleConns != 5 {
		t.Fatalf("pool override = %d/%d, want 50/5", cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	}
}
