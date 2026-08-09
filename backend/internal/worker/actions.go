package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// virusScan validates every stored binary. Without a scanner configured the
// recorded upload checksum is the integrity gate; with a real scanner
// (clamav) each object is fetched and scanned. The step is idempotent.
func (w *Worker) virusScan(ctx context.Context, docID string) error {
	var stores []model.Storage
	if err := w.db.WithContext(ctx).Where("document_id = ?", docID).Find(&stores).Error; err != nil {
		return err
	}
	if len(stores) == 0 {
		logger.Info("worker: virus scan", zap.String("document_id", docID), zap.String("result", "nothing-to-scan"))
		return nil // no binary uploaded
	}
	// Integrity gate: every upload must carry a recorded checksum.
	for _, s := range stores {
		if s.Checksum == "" {
			return w.failSaga(ctx, docID, "virus_scan", "missing checksum")
		}
	}
	if _, ok := w.scanner.(NoopScanner); ok {
		logger.Info("worker: virus scan", zap.String("document_id", docID), zap.String("result", "checksum-only"))
		return nil
	}
	for _, s := range stores {
		if s.ObjectKey == "" || s.Provider == "" {
			continue
		}
		if err := w.scanStoredObject(ctx, docID, s); err != nil {
			return err
		}
	}
	logger.Info("worker: virus scan", zap.String("document_id", docID), zap.String("result", "clean"))
	return nil
}

// scanStoredObject fetches one stored binary and streams it to the scanner.
func (w *Worker) scanStoredObject(ctx context.Context, docID string, s model.Storage) error {
	if w.store == nil {
		return fmt.Errorf("object store not configured for virus scanning")
	}
	key, err := crypto.Decrypt(s.ObjectKey) // stored encrypted at rest
	if err != nil {
		return fmt.Errorf("decrypt object key: %w", err)
	}
	rc, err := w.store.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("fetch object %s: %w", s.ID, err)
	}
	defer func() { _ = rc.Close() }()
	clean, err := w.scanner.Scan(ctx, rc)
	if err != nil {
		// An infected file is a terminal condition (fail the saga); any other
		// scanner error (clamd down, IO) is transient and will be retried.
		if errors.Is(err, ErrVirusFound) {
			return w.failSaga(ctx, docID, "virus_scan", "infected object "+s.ID)
		}
		return err
	}
	if !clean {
		return w.failSaga(ctx, docID, "virus_scan", "infected object "+s.ID)
	}
	return nil
}

// metadataExtraction enriches the document with derived metadata.
func (w *Worker) metadataExtraction(ctx context.Context, docID string) error {
	doc, err := w.docs.GetByID(ctx, docID)
	if err != nil {
		return err
	}
	if doc.Meta == nil {
		doc.Meta = model.JSONMap{}
	}
	meta := doc.Meta
	if _, ok := meta["title_length"]; !ok {
		meta["title_length"] = len(doc.Title)
	}
	if _, ok := meta["tag_count"]; !ok {
		meta["tag_count"] = len(doc.Tags)
	}
	var s model.Storage
	if err := w.db.WithContext(ctx).Where("document_id = ?", docID).First(&s).Error; err == nil {
		meta["size_bytes"] = s.SizeBytes
		meta["mime_type"] = s.MimeType
		meta["checksum"] = s.Checksum
	}
	meta["extracted_at"] = time.Now().UTC().Format(time.RFC3339)
	return w.db.WithContext(ctx).Model(doc).Update("meta", doc.Meta).Error
}

// ensureStorage records a placeholder storage row when no binary was
// uploaded (documents can be registered without a file).
func (w *Worker) ensureStorage(ctx context.Context, docID string) error {
	var count int64
	if err := w.db.WithContext(ctx).Model(&model.Storage{}).
		Where("document_id = ?", docID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now()
	return w.db.WithContext(ctx).Create(&model.Storage{
		DocumentID: docID,
		Provider:   "local",
		FileName:   "",
		Status:     "pending",
		StoredAt:   &now,
	}).Error
}

// indexing marks the document searchable (external search cluster when
// configured) and completes the pipeline.
func (w *Worker) indexing(ctx context.Context, docID string) error {
	doc, err := w.docs.GetByID(ctx, docID)
	if err != nil {
		return err
	}
	if doc.Meta == nil {
		doc.Meta = model.JSONMap{}
	}
	doc.Meta["indexed_at"] = time.Now().UTC().Format(time.RFC3339)
	if err := w.db.WithContext(ctx).Model(doc).Update("meta", doc.Meta).Error; err != nil {
		return err
	}
	if err := w.indexer.Index(ctx, doc); err != nil {
		return fmt.Errorf("index document %s: %w", docID, err)
	}
	return nil
}

// failSaga marks the saga failed and emits a failed event.
func (w *Worker) failSaga(ctx context.Context, docID, step, reason string) error {
	s, err := w.sagas.FindByAggregate(ctx, "document", docID)
	if err != nil {
		return err
	}
	return w.sagas.Fail(ctx, s.ID, step, reason)
}

// enqueue queues an outbox event for publication (same DB as the worker).
func enqueue(ctx context.Context, db *gorm.DB, docID, eventType string, payload any) error {
	return outbox.Enqueue(ctx, db, outbox.Event{
		AggregateType: "document", AggregateID: docID, EventType: eventType,
	}, payload)
}
