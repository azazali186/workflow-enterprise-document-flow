// Package worker consumes the transactional-outbox events published to
// JetStream and executes the document processing pipeline: it advances the
// upload saga step by step (virus scan → metadata extraction → storage →
// indexing), finalises documents, and pushes real-time events to WebSocket
// clients.
package worker

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/objectstore"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/breaker"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/retry"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/trace"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/aeroxe/docu-flow/backend/internal/ws"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SubjectPrefix must match the outbox publisher namespace.
const SubjectPrefix = "docuflow"

// eventPayload mirrors the outbox message JSON published to JetStream.
type eventPayload struct {
	ID            string          `json:"id"`
	TraceID       string          `json:"trace_id,omitempty"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
}

// Worker consumes and processes domain events.
type Worker struct {
	db      *gorm.DB
	js      nats.JetStreamContext
	cache   *cache.Client
	sagas   *saga.Coordinator
	hub     *ws.Hub
	docs    *repository.Repo[model.Document]
	stores  *repository.Repo[model.Storage]
	approve *repository.Repo[model.Approval]
	audit   *service.AuditService
	breaker *breaker.Breaker
	sub     *nats.Subscription
	scanner Scanner
	indexer service.Indexer
	store   objectstore.ObjectStore
}

// Options carries the optional pipeline integrations. When left nil the
// worker falls back to safe defaults (noop scanner, noop indexer).
type Options struct {
	Scanner Scanner
	Indexer service.Indexer
	Store   objectstore.ObjectStore
}

// New wires the worker.
func New(db *gorm.DB, js nats.JetStreamContext, c *cache.Client, sagas *saga.Coordinator,
	hub *ws.Hub, docs *repository.Repo[model.Document], stores *repository.Repo[model.Storage],
	approve *repository.Repo[model.Approval], audit *service.AuditService, opts Options) *Worker {
	if opts.Scanner == nil {
		opts.Scanner = NoopScanner{}
	}
	if opts.Indexer == nil {
		opts.Indexer = NoopIndexer{}
	}
	return &Worker{
		db: db, js: js, cache: c, sagas: sagas, hub: hub,
		docs: docs, stores: stores, approve: approve, audit: audit,
		breaker: breaker.New(breaker.DefaultOptions()),
		scanner: opts.Scanner, indexer: opts.Indexer, store: opts.Store,
	}
}

// Start subscribes to the docuflow stream and blocks until ctx is done.
func (w *Worker) Start(ctx context.Context) error {
	sub, err := w.js.QueueSubscribe(
		SubjectPrefix+".>",
		"docuflow-workers",
		func(m *nats.Msg) { w.process(m) },
		nats.Durable("docuflow-workers"),
		nats.ManualAck(),
		nats.MaxDeliver(5),
		nats.AckWait(30*time.Second),
		nats.DeliverNew(),
	)
	if err != nil {
		return err
	}
	w.sub = sub
	logger.Info("worker: subscribed to docuflow.>")
	<-ctx.Done()
	return w.sub.Unsubscribe()
}

// DLQSubject is where undeliverable events are parked for operator review.
// nats.go v1.37 has no client-side DeadLetter option, so the worker publishes
// terminally-failed raw messages here before terminating the original. The
// "dlq.docuflow." prefix is intentionally disjoint from the main stream's
// "docuflow.>" subjects (JetStream forbids overlapping stream subjects and
// the worker must never consume its own DLQ output).
const DLQSubject = "dlq.docuflow.worker"

// process handles one message with dedupe, retry and breaker semantics.
func (w *Worker) process(m *nats.Msg) {
	var evt eventPayload
	if err := json.Unmarshal(m.Data, &evt); err != nil {
		logger.Warn("worker: undecodable event, sending to DLQ", zap.Error(err))
		w.requeueToDLQ(m)
		_ = m.Term()
		return
	}
	// At-least-once guard: redeliveries of the same message are skipped.
	dedupeKey := "worker:done:" + evt.ID
	ok, err := w.cache.SetNX(dedupeKey, "1")
	if err == nil && !ok {
		_ = m.Ack()
		return
	}
	if err == nil {
		_ = w.cache.Expire(dedupeKey, 24*time.Hour)
	}

	// Carry the originating request id so worker logs and any re-enqueued
	// events stay correlated with the API call that produced them.
	ctx, cancel := context.WithTimeout(trace.WithID(context.Background(), evt.TraceID), 60*time.Second)
	defer cancel()
	logger.Debug("worker: processing event", zap.String("request_id", evt.TraceID),
		zap.String("event", evt.EventType), zap.String("aggregate", evt.AggregateID))
	err = retry.Do(ctx, retry.Options{MaxAttempts: 3}, func(_ int) error {
		return w.breaker.Execute(func() error { return w.dispatch(ctx, evt) })
	})
	switch err {
	case nil:
		_ = m.Ack()
	case context.DeadlineExceeded, context.Canceled:
		_ = m.Nak()
	default:
		logger.Error("worker: event failed permanently", zap.String("id", evt.ID), zap.Error(err))
		w.requeueToDLQ(m)
		_ = m.Term()
	}
}

// requeueToDLQ copies a raw message into the dead-letter stream so a
// permanently failing event is inspectable and replayable instead of being
// silently dropped. Best effort: a DLQ publish failure is only logged.
func (w *Worker) requeueToDLQ(m *nats.Msg) {
	if _, err := w.js.Publish(DLQSubject, m.Data); err != nil {
		logger.Error("worker: dlq publish failed", zap.String("subject", DLQSubject), zap.Error(err))
	}
}

// HandleEvent processes one outbox event through the same dispatch logic the
// NATS subscription uses. Exposed for tests and future admin replay triggers.
func (w *Worker) HandleEvent(ctx context.Context, evt outbox.Event) error {
	payload, err := json.Marshal(evt.Payload)
	if err != nil {
		return err
	}
	return w.dispatch(ctx, eventPayload{
		ID: evt.ID, TraceID: evt.TraceID, AggregateType: evt.AggregateType,
		AggregateID: evt.AggregateID, EventType: evt.EventType, Payload: payload,
	})
}

// dispatch routes an event to its handler by subject suffix.
func (w *Worker) dispatch(ctx context.Context, evt eventPayload) error {
	eventType := strings.TrimPrefix(evt.EventType, "saga.")
	switch eventType {
	case "document_uploaded":
		return w.onDocumentUploaded(ctx, evt)
	case "upload":
		return w.onSagaStep(ctx, evt, w.virusScan)
	case "virus_scan":
		return w.onSagaStep(ctx, evt, w.metadataExtraction)
	case "metadata_extraction":
		return w.onSagaStep(ctx, evt, w.ensureStorage)
	case "storage":
		return w.onSagaStep(ctx, evt, w.indexing)
	case "document_updated", "verification_needed", "verification_rejected",
		"approval_required", "approval_rejected", "document_ready", "access_granted":
		return w.broadcastDocumentEvent(ctx, evt)
	default:
		return nil // saga.started, saga.failed, unknown → ignore
	}
}

// onDocumentUploaded kicks the upload saga by completing its first step.
func (w *Worker) onDocumentUploaded(ctx context.Context, evt eventPayload) error {
	s, err := w.sagas.FindByAggregate(ctx, "document", evt.AggregateID)
	if err != nil {
		return w.broadcastDocumentEvent(ctx, evt)
	}
	if s.Status == saga.StatusFailed {
		return nil // terminal: never re-broadcast for a failed saga
	}
	if s.Status != saga.StatusRunning {
		return w.broadcastDocumentEvent(ctx, evt)
	}
	if _, err := w.sagas.CompleteStep(ctx, s.ID, "upload"); err != nil {
		return err
	}
	return w.broadcastDocumentEvent(ctx, evt)
}

// onSagaStep runs the action for the saga's currently-running step, then
// advances the saga. The event subject (saga.<step>) only signals that a
// transition happened; the next pending step is looked up from saga state.
func (w *Worker) onSagaStep(ctx context.Context, evt eventPayload, action func(context.Context, string) error) error {
	s, err := w.sagas.FindByAggregate(ctx, evt.AggregateType, evt.AggregateID)
	if err != nil {
		return err
	}
	if s.Status == saga.StatusCompleted {
		return w.finalize(ctx, s.AggregateID)
	}
	if s.CurrentStep >= len(s.Steps) {
		return w.finalize(ctx, s.AggregateID)
	}
	if err := action(ctx, s.AggregateID); err != nil {
		return err
	}
	// Reload: the action may have failed the saga (e.g. a virus was found).
	// Failed sagas are terminal — never advance or finalize them.
	s, err = w.sagas.FindByAggregate(ctx, evt.AggregateType, evt.AggregateID)
	if err != nil {
		return err
	}
	if s.Status == saga.StatusFailed {
		return nil
	}
	// Complete the step that is currently running (idempotent on replay).
	stepName := s.Steps[s.CurrentStep].Name
	next, err := w.sagas.CompleteStep(ctx, s.ID, stepName)
	if err != nil {
		return err
	}
	if next.Status == saga.StatusCompleted {
		return w.finalize(ctx, next.AggregateID)
	}
	return nil
}

// finalize transitions a processed document to pending_verification.
func (w *Worker) finalize(ctx context.Context, docID string) error {
	doc, err := w.docs.GetByID(ctx, docID)
	if err != nil {
		return err
	}
	if doc.Status == "draft" {
		if err := w.db.WithContext(ctx).Model(doc).Update("status", "pending_verification").Error; err != nil {
			return err
		}
		doc.Status = "pending_verification"
		_ = w.audit.Change(ctx, service.Actor{ID: "system"}, "document", docID, nil, doc)
		_ = enqueue(ctx, w.db, doc.ID, "verification_needed", map[string]any{
			"document_id": docID, "status": "pending_verification",
		})
	}
	return w.broadcastDocumentEvent(ctx, eventPayload{AggregateID: docID, EventType: "document_ready"})
}

// broadcastDocumentEvent pushes a document event to owner + approvers.
func (w *Worker) broadcastDocumentEvent(ctx context.Context, evt eventPayload) error {
	userIDs, err := w.recipients(ctx, evt.AggregateID)
	if err != nil {
		return err
	}
	w.hub.PublishToUsers(userIDs, ws.Event{Type: evt.EventType, DocumentID: evt.AggregateID})
	return nil
}

// recipients resolves the users interested in a document's lifecycle.
func (w *Worker) recipients(ctx context.Context, docID string) ([]string, error) {
	if docID == "" {
		return nil, nil
	}
	doc, err := w.docs.GetByID(ctx, docID)
	if err != nil {
		if apperror.CodeOf(err) == apperror.CodeNotFound {
			return nil, nil // event without a document → broadcast nothing targeted
		}
		return nil, err
	}
	ids := map[string]bool{doc.OwnerID: true}
	var approvers []model.Approval
	if err := w.db.WithContext(ctx).Where("document_id = ?", docID).
		Distinct("approver_id").Find(&approvers).Error; err == nil {
		for _, a := range approvers {
			ids[a.ApproverID] = true
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out, nil
}
