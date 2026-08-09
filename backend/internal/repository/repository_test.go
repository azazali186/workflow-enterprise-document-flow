package repository

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Category{}, &model.User{}, &model.Role{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedCategories(t *testing.T, db *gorm.DB, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		cat := &model.Category{
			Name:     "Category " + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			Slug:     "cat-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			IsActive: i%2 == 0,
		}
		if err := db.Create(cat).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func listReq(cursor string, limit int) *pagination.Normalized {
	req := &pagination.Request{Limit: limit, Cursor: cursor}
	n, _ := req.Normalize("created_at")
	return n
}

func TestListCursorPaginationWalksAllRows(t *testing.T) {
	db := testDB(t)
	seedCategories(t, db, 25)
	repo := NewCategoryRepo(db)

	ctx := context.Background()
	seen := 0
	cursor := ""
	for page := 1; ; page++ {
		items, meta, _, err := repo.List(ctx, ListQuery{P: listReq(cursor, 10)})
		if err != nil {
			t.Fatal(err)
		}
		seen += len(items)
		t.Logf("page %d: items=%d hasMore=%v total=%d cursor=%q", page, len(items), meta.HasMore, meta.TotalCount, cursor)
		if !meta.HasMore {
			break
		}
		cursor = meta.NextCursor
	}
	if seen != 25 {
		t.Errorf("walked %d rows, want 25", seen)
	}
}

func TestListFilterAndSummary(t *testing.T) {
	db := testDB(t)
	seedCategories(t, db, 20)
	repo := NewCategoryRepo(db)

	n, _ := (&pagination.Request{Limit: 50, Filters: map[string]any{"is_active": true}}).Normalize("created_at")
	items, meta, summary, err := repo.List(context.Background(), ListQuery{P: n})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 10 {
		t.Errorf("active = %d, want 10", len(items))
	}
	if meta.TotalCount != 10 {
		t.Errorf("total = %d, want 10", meta.TotalCount)
	}
	if summary == nil {
		t.Error("summary missing")
	}
	if s, ok := summary["total"].(int64); !ok || s != 20 {
		t.Errorf("summary total = %v, want 20", summary["total"])
	}
}

func TestListDynamicSortByString(t *testing.T) {
	db := testDB(t)
	seedCategories(t, db, 10)
	repo := NewCategoryRepo(db)

	n, _ := (&pagination.Request{Limit: 10, SortBy: "name", SortDir: "asc"}).Normalize("created_at")
	items, _, _, err := repo.List(context.Background(), ListQuery{P: n})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 2 || items[0].Name > items[1].Name {
		t.Errorf("not sorted by name asc: %s > %s", items[0].Name, items[1].Name)
	}
}

func TestListSearch(t *testing.T) {
	db := testDB(t)
	seedCategories(t, db, 10)
	repo := NewCategoryRepo(db)

	n, _ := (&pagination.Request{Limit: 10, Search: "cat-"}).Normalize("created_at")
	items, meta, _, err := repo.List(context.Background(), ListQuery{P: n})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 10 || meta.TotalCount != 10 {
		t.Errorf("search returned %d items / %d total", len(items), meta.TotalCount)
	}
}

func TestListPreloadsRelations(t *testing.T) {
	db := testDB(t)
	role := model.Role{Code: "admin", Name: "Administrator"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{Email: "a@example.com", PasswordHash: "x", Name: "Alice", Status: model.UserActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewUserRepo(db)
	n, _ := (&pagination.Request{Limit: 10}).Normalize("created_at")
	items, _, _, err := repo.List(context.Background(), ListQuery{P: n})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("list returned %d users, want 1", len(items))
	}
	if len(items[0].Roles) != 1 || items[0].Roles[0].Code != "admin" {
		t.Errorf("roles not preloaded: %+v", items[0].Roles)
	}
}

func TestListDateRange(t *testing.T) {
	db := testDB(t)
	seedCategories(t, db, 5)
	repo := NewCategoryRepo(db)

	n, _ := (&pagination.Request{Limit: 10, DateFrom: "2026-01-01"}).Normalize("created_at")
	items, meta, _, err := repo.List(context.Background(), ListQuery{P: n})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 || meta.TotalCount != 5 {
		t.Errorf("date range returned %d/%d", len(items), meta.TotalCount)
	}
	// Range entirely in the past returns nothing.
	n2, _ := (&pagination.Request{Limit: 10, DateFrom: "2000-01-01", DateTo: "2000-01-02"}).Normalize("created_at")
	items2, _, _, err := repo.List(context.Background(), ListQuery{P: n2})
	if err != nil {
		t.Fatal(err)
	}
	if len(items2) != 0 {
		t.Errorf("expected 0 rows in past range, got %d", len(items2))
	}
}
