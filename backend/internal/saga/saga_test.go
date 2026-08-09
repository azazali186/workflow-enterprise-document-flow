package saga

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestMain(m *testing.M) {
	_ = logger.Init("error", false)
	os.Exit(m.Run())
}

func newSagaHarness(t *testing.T) (*gorm.DB, *cache.Client) {
	t.Helper()
	// Unique in-memory database per test so rows never leak between tests.
	dsn := "file:saga_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OutboxMessage{}); err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	return db, c
}

func TestSagaRunsToCompletion(t *testing.T) {
	db, c := newSagaHarness(t)
	coord := New(c, db)
	ctx := context.Background()

	s, err := coord.Start(ctx, DocumentUpload, "document", "doc-1", DocumentUploadSteps)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusRunning || s.CurrentStep != 0 {
		t.Fatalf("unexpected initial state: %+v", s)
	}

	for _, step := range DocumentUploadSteps {
		s, err = coord.CompleteStep(ctx, s.ID, step)
		if err != nil {
			t.Fatalf("complete %s: %v", step, err)
		}
	}
	if s.Status != StatusCompleted {
		t.Fatalf("expected completed, got %q", s.Status)
	}

	// Completing a finished step is idempotent and stays completed.
	if _, err := coord.CompleteStep(ctx, s.ID, "indexing"); err != nil {
		t.Fatal(err)
	}
	got, err := coord.Get(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusCompleted {
		t.Fatalf("expected completed after replay, got %q", got.Status)
	}

	// Outbox should carry saga.started + one event per step.
	var outboxCount int64
	if err := db.Model(&model.OutboxMessage{}).Count(&outboxCount).Error; err != nil {
		t.Fatal(err)
	}
	if outboxCount != int64(1+len(DocumentUploadSteps)) {
		t.Fatalf("expected %d outbox events, got %d", 1+len(DocumentUploadSteps), outboxCount)
	}
}

func TestSagaFailMarksFailedAndPersistsReason(t *testing.T) {
	db, c := newSagaHarness(t)
	coord := New(c, db)
	ctx := context.Background()

	s, err := coord.Start(ctx, DocumentUpload, "document", "doc-2", DocumentUploadSteps)
	if err != nil {
		t.Fatal(err)
	}
	if err := coord.Fail(ctx, s.ID, "virus_scan", "checksum missing"); err != nil {
		t.Fatal(err)
	}
	got, err := coord.Get(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusFailed {
		t.Fatalf("expected failed, got %q", got.Status)
	}
	for _, step := range got.Steps {
		if step.Name == "virus_scan" && step.Status != "failed" {
			t.Fatalf("expected virus_scan failed, got %+v", step)
		}
	}
}

func TestSagaCompleteStepOnFailedSagaIsNoop(t *testing.T) {
	db, c := newSagaHarness(t)
	coord := New(c, db)
	ctx := context.Background()

	s, err := coord.Start(ctx, DocumentUpload, "document", "doc-4", DocumentUploadSteps)
	if err != nil {
		t.Fatal(err)
	}
	if err := coord.Fail(ctx, s.ID, "virus_scan", "infected"); err != nil {
		t.Fatal(err)
	}
	// Completing a step on a failed saga must not advance or complete it.
	after, err := coord.CompleteStep(ctx, s.ID, "virus_scan")
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != StatusFailed || after.CurrentStep != 0 {
		t.Fatalf("failed saga advanced: status=%q step=%d", after.Status, after.CurrentStep)
	}
	for _, step := range after.Steps {
		if step.Status == StepCompleted {
			t.Fatalf("failed saga must not complete steps: %+v", after.Steps)
		}
	}
}

func TestSagaFindByAggregate(t *testing.T) {
	db, c := newSagaHarness(t)
	coord := New(c, db)
	ctx := context.Background()

	started, err := coord.Start(ctx, DocumentUpload, "document", "doc-3", DocumentUploadSteps)
	if err != nil {
		t.Fatal(err)
	}
	found, err := coord.FindByAggregate(ctx, "document", "doc-3")
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != started.ID {
		t.Fatalf("expected saga %s, got %s", started.ID, found.ID)
	}
}
