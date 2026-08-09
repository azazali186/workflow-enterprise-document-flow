package service

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// noopIndexer satisfies the Indexer contract for tests.
type noopIndexer struct{}

func (noopIndexer) Index(context.Context, *model.Document) error { return nil }
func (noopIndexer) Delete(context.Context, string) error         { return nil }

type docAccessHarness struct {
	db      *gorm.DB
	svc     DocumentService
	search  SearchService
	admin   *model.User
	owner   *model.User
	other   *model.User
	grantee *model.User
}

func newDocAccessHarness(t *testing.T) *docAccessHarness {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{
		&model.User{}, &model.Role{}, &model.UserRole{},
		&model.Document{}, &model.Version{}, &model.Access{},
		&model.AuditLog{}, &model.OutboxMessage{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	audit := NewAuditService(db)
	co := saga.New(c, db)
	svc := NewDocumentService(db, repository.NewDocumentRepo(db), repository.NewVersionRepo(db), audit, c, co, noopIndexer{})
	search := NewSearchService(db, repository.NewDocumentRepo(db))

	return &docAccessHarness{
		db:      db,
		svc:     svc,
		search:  search,
		admin:   createUserWithRole(t, db, "admin@example.com", constant.RoleSuperAdmin),
		owner:   createUserWithRole(t, db, "owner@example.com", constant.RoleUser),
		other:   createUserWithRole(t, db, "other@example.com", constant.RoleUser),
		grantee: createUserWithRole(t, db, "grantee@example.com", constant.RoleUser),
	}
}

func createUserWithRole(t *testing.T, db *gorm.DB, email, roleCode string) *model.User {
	t.Helper()
	user := &model.User{Email: email, PasswordHash: "x", Status: model.UserActive}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	// Roles are unique by code; reuse the row when a previous user already
	// created it in this harness's database.
	var role model.Role
	if err := db.Where("code = ?", roleCode).First(&role).Error; err != nil {
		role = model.Role{Code: roleCode, Name: roleCode, IsSystem: true}
		if err := db.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func (h *docAccessHarness) seedDoc(t *testing.T, ownerID, title string) *model.Document {
	t.Helper()
	doc := &model.Document{
		DocumentNumber: "DOC-" + uuidx.New(),
		Title:          title,
		OwnerID:        ownerID,
		Status:         constant.DocDraft,
	}
	if err := h.db.Create(doc).Error; err != nil {
		t.Fatal(err)
	}
	return doc
}

func (h *docAccessHarness) grantUser(t *testing.T, docID, userID, permission string) {
	t.Helper()
	g := &model.Access{DocumentID: docID, UserID: model.NullableString(userID), Permission: permission, GrantedBy: h.admin.ID}
	if err := h.db.Create(g).Error; err != nil {
		t.Fatal(err)
	}
}

func (h *docAccessHarness) grantRole(t *testing.T, docID, roleID, permission string) {
	t.Helper()
	g := &model.Access{DocumentID: docID, RoleID: model.NullableString(roleID), Permission: permission, GrantedBy: h.admin.ID}
	if err := h.db.Create(g).Error; err != nil {
		t.Fatal(err)
	}
}

func isNotFound(err error) bool { return err != nil && apperror.CodeOf(err) == apperror.CodeNotFound }

func TestDocumentGetEnforcesAccess(t *testing.T) {
	h := newDocAccessHarness(t)
	doc := h.seedDoc(t, h.owner.ID, "Confidential")

	ctx := context.Background()
	if _, err := h.svc.Get(ctx, Actor{ID: h.owner.ID}, doc.ID); err != nil {
		t.Fatalf("owner read denied: %v", err)
	}
	if _, err := h.svc.Get(ctx, Actor{ID: h.other.ID}, doc.ID); !isNotFound(err) {
		t.Fatalf("stranger read must be 404, got %v", err)
	}
	if _, err := h.svc.Get(ctx, Actor{ID: h.admin.ID}, doc.ID); err != nil {
		t.Fatalf("super admin read denied: %v", err)
	}
	// Active grant (read) unlocks the document.
	h.grantUser(t, doc.ID, h.grantee.ID, "read")
	if _, err := h.svc.Get(ctx, Actor{ID: h.grantee.ID}, doc.ID); err != nil {
		t.Fatalf("granted user read denied: %v", err)
	}
	// Revoking the grant closes access again.
	now := time.Now()
	if err := h.db.Model(&model.Access{}).Where("document_id = ? AND user_id = ?", doc.ID, h.grantee.ID).Update("revoked_at", now).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Get(ctx, Actor{ID: h.grantee.ID}, doc.ID); !isNotFound(err) {
		t.Fatalf("revoked grant must deny read, got %v", err)
	}
}

func TestDocumentGetAllowsRoleGrant(t *testing.T) {
	h := newDocAccessHarness(t)
	doc := h.seedDoc(t, h.owner.ID, "Role protected")

	var role model.Role
	if err := h.db.Where("code = ?", constant.RoleUser).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	h.grantRole(t, doc.ID, role.ID, "read")

	if _, err := h.svc.Get(context.Background(), Actor{ID: h.grantee.ID}, doc.ID); err != nil {
		t.Fatalf("role-granted user read denied: %v", err)
	}
}

func TestDocumentListScopesByAccess(t *testing.T) {
	h := newDocAccessHarness(t)
	own := h.seedDoc(t, h.owner.ID, "Own doc")
	_ = h.seedDoc(t, h.other.ID, "Other doc")
	h.grantUser(t, own.ID, h.grantee.ID, "read")

	ctx := context.Background()
	n, err := (&pagination.Request{Limit: 50}).Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	items, meta, _, err := h.svc.List(ctx, Actor{ID: h.owner.ID}, n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].OwnerID != h.owner.ID {
		t.Fatalf("owner list = %d items, want only own; got %+v", len(items), items)
	}
	if meta.TotalCount != 1 {
		t.Fatalf("owner total = %d, want 1", meta.TotalCount)
	}
	items, _, _, err = h.svc.List(ctx, Actor{ID: h.grantee.ID}, n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != own.ID {
		t.Fatalf("grantee list = %d items, want the granted doc; got %+v", len(items), items)
	}
	// Super admin sees everything.
	items, _, _, err = h.svc.List(ctx, Actor{ID: h.admin.ID}, n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("super admin list = %d items, want 2", len(items))
	}
	// Unknown identity sees nothing.
	items, _, _, err = h.svc.List(ctx, Actor{ID: "00000000-0000-0000-0000-000000000000"}, n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("unrelated user list = %d items, want 0", len(items))
	}
}

func TestDocumentUpdateRequiresModifyGrant(t *testing.T) {
	h := newDocAccessHarness(t)
	doc := h.seedDoc(t, h.owner.ID, "Editable")

	ctx := context.Background()
	// Stranger with no grant: denied like a missing row.
	if _, err := h.svc.Update(ctx, Actor{ID: h.other.ID}, UpdateDocumentInput{ID: doc.ID, Title: "hacked"}); !isNotFound(err) {
		t.Fatalf("stranger update must be 404, got %v", err)
	}
	// Read-only grant still cannot modify.
	h.grantUser(t, doc.ID, h.grantee.ID, "read")
	if _, err := h.svc.Update(ctx, Actor{ID: h.grantee.ID}, UpdateDocumentInput{ID: doc.ID, Title: "still locked"}); !isNotFound(err) {
		t.Fatalf("read-only grant update must be 404, got %v", err)
	}
	// Write grant can modify.
	h.grantUser(t, doc.ID, h.grantee.ID, "write")
	updated, err := h.svc.Update(ctx, Actor{ID: h.grantee.ID}, UpdateDocumentInput{ID: doc.ID, Title: "updated by grantee"})
	if err != nil {
		t.Fatalf("write-grant update denied: %v", err)
	}
	if updated.Title != "updated by grantee" {
		t.Fatalf("title not updated: %q", updated.Title)
	}
	// Owner can always modify.
	if _, err := h.svc.Update(ctx, Actor{ID: h.owner.ID}, UpdateDocumentInput{ID: doc.ID, Title: "updated by owner"}); err != nil {
		t.Fatalf("owner update denied: %v", err)
	}
}

func TestDocumentDeleteRequiresModifyGrant(t *testing.T) {
	h := newDocAccessHarness(t)
	doc := h.seedDoc(t, h.owner.ID, "Deletable")

	ctx := context.Background()
	if err := h.svc.Delete(ctx, Actor{ID: h.other.ID}, doc.ID); !isNotFound(err) {
		t.Fatalf("stranger delete must be 404, got %v", err)
	}
	h.grantUser(t, doc.ID, h.grantee.ID, "write")
	if err := h.svc.Delete(ctx, Actor{ID: h.grantee.ID}, doc.ID); err != nil {
		t.Fatalf("write-grant delete denied: %v", err)
	}
}

func TestDocumentSearchScopedByAccess(t *testing.T) {
	h := newDocAccessHarness(t)
	own := h.seedDoc(t, h.owner.ID, "Searchable Alpha")
	_ = h.seedDoc(t, h.other.ID, "Searchable Beta")
	h.grantUser(t, own.ID, h.grantee.ID, "read")

	ctx := context.Background()
	n, err := (&pagination.Request{Limit: 50, Search: "searchable"}).Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	items, meta, err := h.search.SearchDocuments(ctx, h.grantee.ID, n)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != own.ID {
		t.Fatalf("grantee search = %d items, want only the granted doc; got %+v", len(items), items)
	}
	if meta.TotalCount != 1 {
		t.Fatalf("grantee search total = %d, want 1", meta.TotalCount)
	}
}

