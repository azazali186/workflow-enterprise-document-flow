package service

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newOptionsTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	// A unique file name per test: sqlite's shared in-memory cache means two
	// tests opening "file:options" would share one database and leak rows.
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{
		&model.User{}, &model.Role{}, &model.Category{}, &model.Template{}, &model.Document{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedOptions(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Create(&[]model.User{
		{Email: "alice@example.com", Name: "Alice Adams", Status: model.UserActive},
		{Email: "bob@example.com", Name: "Bob Brown", Status: model.UserActive},
		{Email: "locked@example.com", Name: "Locked Larry", Status: model.UserLocked},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.Category{
		{Name: "Finance", Slug: "finance"},
		{Name: "Legal", Slug: "legal"},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.Document{
		{DocumentNumber: "DOC-001", Title: "Annual Report", OwnerID: "u1"},
		{DocumentNumber: "DOC-002", Title: "Tax Filing", OwnerID: "u1"},
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestOptionsListUsersOnlyActiveSorted(t *testing.T) {
	db := newOptionsTestDB(t, "optusers")
	seedOptions(t, db)
	svc := NewOptionsService(db)

	opts, err := svc.List(context.Background(), "users", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 2 {
		t.Fatalf("expected 2 active users, got %d: %+v", len(opts), opts)
	}
	if opts[0].Name != "Alice Adams" || opts[0].ID == "" {
		t.Fatalf("unexpected first option: %+v", opts[0])
	}
}

func TestOptionsListDocumentsUseTitle(t *testing.T) {
	db := newOptionsTestDB(t, "optdocs")
	seedOptions(t, db)
	svc := NewOptionsService(db)

	opts, err := svc.List(context.Background(), "documents", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 2 || opts[0].Name != "Annual Report" {
		t.Fatalf("unexpected document options: %+v", opts)
	}
}

func TestOptionsSearchIsCaseInsensitiveAndMatchesEmail(t *testing.T) {
	db := newOptionsTestDB(t, "optsearch")
	seedOptions(t, db)
	svc := NewOptionsService(db)

	byEmail, err := svc.List(context.Background(), "users", "ALICE@", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(byEmail) != 1 || byEmail[0].Name != "Alice Adams" {
		t.Fatalf("email search failed: %+v", byEmail)
	}

	byName, err := svc.List(context.Background(), "users", "brown", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(byName) != 1 || byName[0].Name != "Bob Brown" {
		t.Fatalf("case-insensitive name search failed: %+v", byName)
	}
}

func TestOptionsUnknownKind(t *testing.T) {
	db := newOptionsTestDB(t, "optbad")
	svc := NewOptionsService(db)

	_, err := svc.List(context.Background(), "gadgets", "", 20)
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if code := apperror.CodeOf(err); code != 400 {
		t.Fatalf("expected 400, got %d", code)
	}
}

func TestOptionsLimitCapped(t *testing.T) {
	db := newOptionsTestDB(t, "optlimit")
	seedOptions(t, db)
	svc := NewOptionsService(db)

	// A huge requested limit is clamped to 50; results are still capped by the
	// actual row count.
	opts, err := svc.List(context.Background(), "users", "", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) > 2 {
		t.Fatalf("limit not honored: got %d rows", len(opts))
	}
}
