// Package database owns the shared infrastructure connections (Postgres via
// GORM, Redis, NATS) and the RBAC seed. All persistence goes through GORM —
// there is no raw SQL anywhere in the codebase.
package database

import (
	"context"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB is the process-wide GORM handle.
var DB *gorm.DB

// models lists every entity migrated at startup.
var models = []any{
	&model.User{},
	&model.Role{},
	&model.Permission{},
	&model.Document{},
	&model.Category{},
	&model.Version{},
	&model.Verification{},
	&model.Approval{},
	&model.Storage{},
	&model.Template{},
	&model.Access{},
	&model.LoginLog{},
	&model.AuditLog{},
	&model.OutboxMessage{},
}

// Init opens PostgreSQL through GORM and runs migrations.
func Init(cfg *config.Config) (*gorm.DB, error) {
	level := gormlogger.Warn
	if cfg.Env == "development" {
		level = gormlogger.Info
	}
	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(level),
	}
	if cfg.IsProduction() {
		gormCfg.Logger = gormlogger.New(
			&gormZapWriter{},
			gormlogger.Config{SlowThreshold: 500 * time.Millisecond, LogLevel: gormlogger.Warn},
		)
	}
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), gormCfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// Pool sizes are configurable (DB_MAX_OPEN_CONNS / DB_MAX_IDLE_CONNS) so
	// operators can size the pool per replica against PostgreSQL's
	// max_connections instead of inheriting a fixed default. Note: with the
	// HPA scaling to N replicas the aggregate connections are pool × replicas.
	maxOpen := cfg.DBMaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.DBMaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	DB = db
	return db, nil
}

// Migrate applies schema migrations for all entities (idempotent).
func Migrate(db *gorm.DB) error {
	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			logger.Error("auto-migrate failed", zap.Any("model", m), zap.Error(err))
			return err
		}
	}
	return nil
}

// Ping verifies DB connectivity.
func Ping(ctx context.Context) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// gormZapWriter forwards GORM slow logs into zap.
type gormZapWriter struct{}

// Printf implements gormlogger.Writer.
func (g *gormZapWriter) Printf(format string, args ...any) {
	logger.S.Debugf(format, args...)
}
