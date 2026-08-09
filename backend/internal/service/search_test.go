package service

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func searchHarness(t *testing.T) (SearchService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	// The search service applies the document access scope, which resolves
	// ownership/grants through users, roles and accesses — migrate them too.
	for _, m := range []any{
		&model.Document{}, &model.Category{},
		&model.User{}, &model.Role{}, &model.UserRole{}, &model.Access{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	return NewSearchService(db, repository.NewDocumentRepo(db)), db
}

func seedSearchDoc(t *testing.T, db *gorm.DB, title, status string) *model.Document {
	t.Helper()
	doc := &model.Document{
		DocumentNumber: "DOC-SEARCH-" + title,
		Title:          title,
		Description:    "search fixture for " + title,
		OwnerID:        "00000000-0000-0000-0000-000000000001",
		Status:         status,
	}
	if err := db.Create(doc).Error; err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestSearchDocumentsByKeyword(t *testing.T) {
	svc, db := searchHarness(t)
	seedSearchDoc(t, db, "Employment Contract 2026", "draft")
	seedSearchDoc(t, db, "Lease Agreement", "draft")
	seedSearchDoc(t, db, "Vendor Invoice", "verified")

	n, err := (&pagination.Request{Limit: 10, Search: "contract"}).Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	items, meta, err := svc.SearchDocuments(context.Background(), "00000000-0000-0000-0000-000000000001", n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d results, want 1", len(items))
	}
	if meta.TotalCount != 1 {
		t.Fatalf("total = %d, want 1", meta.TotalCount)
	}
	if items[0].Title != "Employment Contract 2026" {
		t.Fatalf("unexpected hit %q", items[0].Title)
	}
}

func TestSearchDocumentsFilterByStatus(t *testing.T) {
	svc, db := searchHarness(t)
	seedSearchDoc(t, db, "Draft Memo", "draft")
	seedSearchDoc(t, db, "Verified Memo", "verified")

	n, err := (&pagination.Request{
		Limit: 10, Search: "memo",
		Filters: map[string]any{"status": "verified"},
	}).Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := svc.SearchDocuments(context.Background(), "00000000-0000-0000-0000-000000000001", n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Verified Memo" {
		t.Fatalf("status filter failed: %+v", items)
	}
}

func TestSearchDocumentsRequiresTerm(t *testing.T) {
	svc, _ := searchHarness(t)
	n, err := (&pagination.Request{Limit: 10}).Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.SearchDocuments(context.Background(), "00000000-0000-0000-0000-000000000001", n)
	if err == nil {
		t.Fatal("expected an error for an empty search term")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeBadRequest {
		t.Fatalf("code = %d, want %d", code, apperror.CodeBadRequest)
	}
}
