package service

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func storageHarness(t *testing.T) (StorageService, *gorm.DB) {
	t.Helper()
	// Register encrypts the object key at rest; initialise the (dev) AEAD key.
	if err := crypto.Init(""); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{&model.Document{}, &model.Storage{}, &model.AuditLog{}} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewStorageService(repository.NewStorageRepo(db), repository.NewDocumentRepo(db), NewAuditService(db))
	return svc, db
}

func seedStorageDoc(t *testing.T, db *gorm.DB) *model.Document {
	t.Helper()
	doc := &model.Document{
		DocumentNumber: "DOC-STORAGE-1",
		Title:          "storage fixture",
		OwnerID:        "00000000-0000-0000-0000-000000000001",
		Status:         constant.DocDraft,
	}
	if err := db.Create(doc).Error; err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestRegisterRejectsUnsafeObjectKey(t *testing.T) {
	svc, db := storageHarness(t)
	doc := seedStorageDoc(t, db)
	ctx := context.Background()

	unsafe := []string{
		"../../etc/passwd",     // traversal out of the storage root
		"/etc/passwd",          // absolute path
		"a/../../b/secret.pdf", // traversal inside the key
		"..",                   // bare parent
		"safe/../../escape",    // leading safe segment, then escape
	}
	for _, key := range unsafe {
		_, err := svc.Register(ctx, Actor{}, StorageRegisterInput{
			DocumentID: doc.ID, ObjectKey: key, Provider: "s3",
		})
		if err == nil || apperror.CodeOf(err) != apperror.CodeBadRequest {
			t.Fatalf("object_key %q must be rejected with 400, got %v", key, err)
		}
	}
}

func TestRegisterAcceptsSafeObjectKey(t *testing.T) {
	svc, db := storageHarness(t)
	doc := seedStorageDoc(t, db)
	ctx := context.Background()

	for _, key := range []string{"dir/file.pdf", "file.pdf", "nested/deeper/object.bin", ""} {
		rec, err := svc.Register(ctx, Actor{}, StorageRegisterInput{
			DocumentID: doc.ID, ObjectKey: key, Provider: "s3", FileName: "f", SizeBytes: 1,
		})
		if err != nil {
			t.Fatalf("safe object_key %q rejected: %v", key, err)
		}
		if key != "" && (rec.ObjectKey == "" || rec.ObjectKey == key) {
			t.Fatalf("object key must be encrypted at rest for %q, got %q", key, rec.ObjectKey)
		}
	}
}
