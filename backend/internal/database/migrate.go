package database

import (
	"embed"
	"errors"
	"fmt"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateVersioned applies versioned SQL migrations. Postgres uses the
// embedded golang-migrate files (reviewable, reversible); any other dialector
// (sqlite in tests) falls back to AutoMigrate.
func MigrateVersioned(db *gorm.DB, cfg *config.Config) error {
	if db.Name() != "postgres" {
		return Migrate(db)
	}
	m, err := NewMigrator(cfg)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if _, err := m.Close(); err != nil {
		return fmt.Errorf("close migrator: %w", err)
	}
	return nil
}

// NewMigrator builds a golang-migrate instance backed by the embedded SQL
// files and the configured Postgres URL. Callers own Close().
func NewMigrator(cfg *config.Config) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open migrator: %w", err)
	}
	return m, nil
}
