package database

import (
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestMigrateVersionedFallsBackToAutoMigrateForSqlite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migrate_test?mode=memory&cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{} // only used for postgres; sqlite path ignores it

	if err := MigrateVersioned(db, cfg); err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{&model.User{}, &model.OutboxMessage{}, &model.Document{}} {
		if !db.Migrator().HasTable(m) {
			t.Fatalf("expected table for %T after fallback migration", m)
		}
	}
}
