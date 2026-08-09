// migrate applies versioned schema migrations and the RBAC seed against the
// configured database without starting the server.
//
//	go run cmd/migrate/main.go up
//	go run cmd/migrate/main.go down
//	go run cmd/migrate/main.go version
//	go run cmd/migrate/main.go force <version>
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/database"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]

	// Migrations only need DATABASE_URL, not the full web-app config.
	cfg, err := config.LoadMigrations()
	logger.FatalIf(err, "load config")
	logger.FatalIf(logger.Init(cfg.LogLevel, false), "init logger")

	m, err := database.NewMigrator(cfg)
	logger.FatalIf(err, "open migrator")

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.Fatal("migrate up", zap.Error(err))
		}
		// Seed built-in roles and the bootstrap admin (idempotent).
		db, err := database.Init(cfg)
		logger.FatalIf(err, "connect database")
		if err := database.SeedRBAC(db, cfg); err != nil {
			logger.Fatal("seed rbac", zap.Error(err))
		}
		sqlDB, err := db.DB()
		logger.FatalIf(err, "db handle")
		logger.FatalIf(sqlDB.Close(), "close database")
		logger.Info("migrations applied and RBAC seeded")
	case "down":
		if err := m.Steps(-1); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("already at the base schema (no migration to roll back)")
				return
			}
			logger.Fatal("migrate down", zap.Error(err))
		}
		logger.Info("rolled back one migration")
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("no migrations applied")
				return
			}
			logger.Fatal("read version", zap.Error(err))
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
	case "force":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[2])
		logger.FatalIf(err, "version must be an integer")
		logger.FatalIf(m.Force(version), "force version")
		logger.Info("forced version", zap.Int("version", version))
	default:
		usage()
		os.Exit(1)
	}
	_, _ = m.Close()
}

func usage() {
	fmt.Println(`usage: go run cmd/migrate/main.go <command>

commands:
  up            apply all pending migrations and seed RBAC
  down          roll back one migration
  version       print the current schema version
  force <v>     set the schema version without running migrations`)
}
