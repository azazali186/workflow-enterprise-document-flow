package worker

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/aeroxe/docu-flow/backend/internal/ws"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newTestWorker wires a worker with an in-memory sqlite DB + miniredis so the
// saga state machine and step actions run without external services.
func newTestWorker(t *testing.T) (*Worker, *gorm.DB, *saga.Coordinator) {
	t.Helper()
	_ = logger.Init("warn", false)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, m := range []any{
		&model.Document{}, &model.Storage{}, &model.OutboxMessage{}, &model.AuditLog{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatalf("migrate %T: %v", m, err)
		}
	}

	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	co := saga.New(c, db)
	audit := service.NewAuditService(db)
	w := New(db, nil, c, co, ws.NewHub(),
		repository.NewDocumentRepo(db), repository.NewStorageRepo(db),
		repository.NewApprovalRepo(db), audit, Options{})
	return w, db, co
}

func seedDocument(t *testing.T, db *gorm.DB, id, status string) *model.Document {
	t.Helper()
	doc := &model.Document{
		BaseModel: model.BaseModel{ID: id},
		Title:     "Test Doc",
		Status:    status,
		OwnerID:   "10000000-0000-7000-8000-000000000001",
		Meta:      model.JSONMap{},
	}
	if err := db.Create(doc).Error; err != nil {
		t.Fatalf("seed document: %v", err)
	}
	return doc
}

func TestWorkerRunsUploadPipeline(t *testing.T) {
	w, db, co := newTestWorker(t)
	ctx := context.Background()
	docID := "20000000-0000-7000-8000-000000000001"
	seedDocument(t, db, docID, "draft")

	if _, err := co.Start(ctx, saga.DocumentUpload, "document", docID, saga.DocumentUploadSteps); err != nil {
		t.Fatalf("start saga: %v", err)
	}

	// The worker receives document_uploaded then one event per saga step.
	if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: "document_uploaded"}); err != nil {
		t.Fatalf("document_uploaded: %v", err)
	}
	for _, evt := range []string{"saga.upload", "saga.virus_scan", "saga.metadata_extraction", "saga.storage"} {
		if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: evt}); err != nil {
			t.Fatalf("%s: %v", evt, err)
		}
	}

	var doc model.Document
	if err := db.First(&doc, "id = ?", docID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if doc.Status != "pending_verification" {
		t.Fatalf("status = %s, want pending_verification", doc.Status)
	}
	if doc.Meta["indexed_at"] == nil || doc.Meta["title_length"] == nil {
		t.Fatalf("metadata not enriched: %+v", doc.Meta)
	}
	// A storage placeholder row must exist (no binary uploaded).
	var count int64
	if err := db.Model(&model.Storage{}).Where("document_id = ?", docID).Count(&count).Error; err != nil {
		t.Fatalf("count storage: %v", err)
	}
	if count != 1 {
		t.Fatalf("storage rows = %d, want 1", count)
	}
	final, err := co.FindByAggregate(ctx, "document", docID)
	if err != nil || final.Status != saga.StatusCompleted {
		t.Fatalf("saga not completed: %+v err=%v", final, err)
	}
}

func TestWorkerIdempotentReplay(t *testing.T) {
	w, db, co := newTestWorker(t)
	ctx := context.Background()
	docID := "20000000-0000-7000-8000-000000000002"
	seedDocument(t, db, docID, "draft")

	if _, err := co.Start(ctx, saga.DocumentUpload, "document", docID, saga.DocumentUploadSteps); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Replaying the same event twice must not double-advance the saga.
	for i := 0; i < 2; i++ {
		if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: "document_uploaded"}); err != nil {
			t.Fatalf("replay %d: %v", i, err)
		}
	}
	s, err := co.FindByAggregate(ctx, "document", docID)
	if err != nil {
		t.Fatalf("find saga: %v", err)
	}
	if s.CurrentStep != 1 {
		t.Fatalf("after replay current_step = %d, want 1 (only advanced once)", s.CurrentStep)
	}
}

func TestWorkerVirusScanFailsWithoutChecksum(t *testing.T) {
	w, db, co := newTestWorker(t)
	ctx := context.Background()
	docID := "20000000-0000-7000-8000-000000000003"
	seedDocument(t, db, docID, "draft")
	// A storage row with an empty checksum is a failed virus scan.
	if err := db.Create(&model.Storage{DocumentID: docID, Provider: "local", Status: "stored"}).Error; err != nil {
		t.Fatalf("seed storage: %v", err)
	}

	if _, err := co.Start(ctx, saga.DocumentUpload, "document", docID, saga.DocumentUploadSteps); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: "document_uploaded"}); err != nil {
		t.Fatalf("document_uploaded: %v", err)
	}
	// saga.upload runs virusScan → no checksum → saga fails.
	if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: "saga.upload"}); err != nil {
		t.Fatalf("saga.upload: %v", err)
	}
	s, err := co.FindByAggregate(ctx, "document", docID)
	if err != nil || s.Status != saga.StatusFailed {
		t.Fatalf("saga should be failed: %+v err=%v", s, err)
	}
	// Document stays draft after failure.
	var doc model.Document
	_ = db.First(&doc, "id = ?", docID)
	if doc.Status != "draft" {
		t.Fatalf("status = %s, want draft after failed saga", doc.Status)
	}
}

func TestWorkerIgnoresUnknownEvents(t *testing.T) {
	w, _, _ := newTestWorker(t)
	if err := w.HandleEvent(context.Background(), outbox.Event{
		AggregateType: "saga", AggregateID: "x", EventType: "saga.started",
	}); err != nil {
		t.Fatalf("unknown event should be a no-op, got %v", err)
	}
}

func TestWorkerMetadataExtractionUsesStorage(t *testing.T) {
	w, db, _ := newTestWorker(t)
	ctx := context.Background()
	docID := "20000000-0000-7000-8000-000000000004"
	seedDocument(t, db, docID, "draft")
	now := time.Now()
	if err := db.Create(&model.Storage{
		DocumentID: docID, Provider: "s3", FileName: "a.pdf", MimeType: "application/pdf",
		SizeBytes: 1234, Checksum: "abc", Status: "stored", StoredAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed storage: %v", err)
	}
	if err := w.metadataExtraction(ctx, docID); err != nil {
		t.Fatalf("metadataExtraction: %v", err)
	}
	var doc model.Document
	if err := db.First(&doc, "id = ?", docID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if doc.Meta["size_bytes"] != float64(1234) || doc.Meta["mime_type"] != "application/pdf" {
		t.Fatalf("metadata missing storage info: %+v", doc.Meta)
	}
}
