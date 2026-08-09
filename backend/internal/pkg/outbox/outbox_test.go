package outbox

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/lock"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/trace"
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

func newOutboxHarness(t *testing.T) (*gorm.DB, *lock.Lock) {
	t.Helper()
	// Unique in-memory database per test so rows never leak between tests.
	dsn := "file:outbox_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
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
	lk := lock.New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	return db, lk
}

type fakePublisher struct {
	subjects []string
	fail     bool
}

func (f *fakePublisher) Publish(subject string, _ []byte) error {
	if f.fail {
		return errors.New("jetstream unavailable")
	}
	f.subjects = append(f.subjects, subject)
	return nil
}

func TestDispatchPublishesPendingMessages(t *testing.T) {
	db, lk := newOutboxHarness(t)
	pub := &fakePublisher{}
	d := NewDispatcher(db, pub, Options{Interval: time.Minute, Batch: 10, MaxAttempts: 3})
	// The request trace id must ride along onto the stored outbox rows.
	ctx := trace.WithID(context.Background(), "trace-abc-123")

	for i := 0; i < 2; i++ {
		if err := Enqueue(ctx, db, Event{
			AggregateType: "document", AggregateID: "doc-1", EventType: "document_uploaded",
		}, map[string]any{"i": i}); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.DispatchOnce(ctx, lk); err != nil {
		t.Fatal(err)
	}

	var msgs []model.OutboxMessage
	if err := db.Find(&msgs).Error; err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	for _, m := range msgs {
		if m.Status != StatusPublished {
			t.Fatalf("expected published, got %q", m.Status)
		}
		if m.TraceID != "trace-abc-123" {
			t.Fatalf("trace id not propagated to outbox row, got %q", m.TraceID)
		}
	}
	if len(pub.subjects) != 2 {
		t.Fatalf("expected 2 publishes, got %d", len(pub.subjects))
	}
	if pub.subjects[0] != "docuflow.document_uploaded" {
		t.Fatalf("unexpected subject %q", pub.subjects[0])
	}
}

func TestDispatchMarksFailedAfterMaxAttempts(t *testing.T) {
	db, lk := newOutboxHarness(t)
	d := NewDispatcher(db, &fakePublisher{fail: true}, Options{Interval: time.Minute, Batch: 10, MaxAttempts: 3})
	ctx := context.Background()

	if err := Enqueue(ctx, db, Event{
		AggregateType: "document", AggregateID: "doc-2", EventType: "document_updated",
	}, map[string]any{}); err != nil {
		t.Fatal(err)
	}

	// One cycle leaves the message pending with attempts incremented.
	if err := d.DispatchOnce(ctx, lk); err != nil {
		t.Fatal(err)
	}
	var m model.OutboxMessage
	if err := db.First(&m).Error; err != nil {
		t.Fatal(err)
	}
	if m.Status != StatusPending {
		t.Fatalf("expected pending after first failure, got %q", m.Status)
	}
	if m.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", m.Attempts)
	}

	// After enough failed cycles the message is parked as failed.
	if err := d.DispatchOnce(ctx, lk); err != nil {
		t.Fatal(err)
	}
	if err := d.DispatchOnce(ctx, lk); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&m).Error; err != nil {
		t.Fatal(err)
	}
	if m.Status != StatusFailed {
		t.Fatalf("expected failed, got %q", m.Status)
	}
}

func TestDispatchIsIdempotentForEmptyQueue(t *testing.T) {
	db, lk := newOutboxHarness(t)
	d := NewDispatcher(db, &fakePublisher{}, Options{Interval: time.Minute, Batch: 10, MaxAttempts: 3})
	if err := d.DispatchOnce(context.Background(), lk); err != nil {
		t.Fatalf("empty dispatch should succeed, got %v", err)
	}
}
