// Package outbox implements the transactional outbox pattern: domain events
// are persisted in the same DB transaction as the business change, then a
// background dispatcher publishes them to NATS JetStream with retry/backoff
// guarded by a circuit breaker.
package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/breaker"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/lock"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/retry"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/trace"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Message statuses.
const (
	StatusPending   = "pending"
	StatusPublished = "published"
	StatusFailed    = "failed"
)

// SubjectPrefix is the NATS subject namespace.
const SubjectPrefix = "docuflow"

// Event is a domain event queued for publication.
type Event struct {
	ID            string    `json:"id"`
	TraceID       string    `json:"trace_id,omitempty"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	Payload       any       `json:"payload"`
}

// Enqueue inserts a pending message using tx, so it commits atomically with
// the business write. tx may be nil to use the plain db handle. The request
// trace id (when present in ctx) is carried on the event for end-to-end
// correlation in the worker.
func Enqueue(ctx context.Context, db *gorm.DB, evt Event, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if evt.TraceID == "" {
		evt.TraceID = trace.ID(ctx)
	}
	now := time.Now()
	msg := model.OutboxMessage{
		BaseModel:     model.BaseModel{ID: uuidx.New(), CreatedAt: now, UpdatedAt: now},
		TraceID:       evt.TraceID,
		AggregateType: evt.AggregateType,
		AggregateID:   evt.AggregateID,
		EventType:     evt.EventType,
		Payload:       string(raw),
		Status:        StatusPending,
	}
	return db.WithContext(ctx).Create(&msg).Error
}

// Dispatcher consumes pending messages and publishes them to JetStream.
type Dispatcher struct {
	db       *gorm.DB
	js       Publisher
	breaker  *breaker.Breaker
	interval time.Duration
	batch    int
	maxTry   int
	subject  func(evtType string) string
}

// Publisher is the minimal JetStream interface the dispatcher needs.
type Publisher interface {
	Publish(subject string, data []byte) error
}

// NewDispatcher wires the dispatcher.
func NewDispatcher(db *gorm.DB, js Publisher, opts Options) *Dispatcher {
	return &Dispatcher{
		db:       db,
		js:       js,
		breaker:  breaker.New(breaker.DefaultOptions()),
		interval: opts.Interval,
		batch:    opts.Batch,
		maxTry:   opts.MaxAttempts,
		subject: func(evtType string) string { return SubjectPrefix + "." + evtType },
	}
}

// Options configures the dispatcher loop.
type Options struct {
	Interval    time.Duration
	Batch       int
	MaxAttempts int
}

// Run starts the dispatch loop and blocks until ctx is done.
func (d *Dispatcher) Run(ctx context.Context, lk *lock.Lock) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.DispatchOnce(ctx, lk); err != nil {
				logger.Error("outbox dispatch cycle failed", zap.Error(err))
			}
		}
	}
}

// DispatchOnce runs one dispatch cycle (exported for tests and admin replay).
func (d *Dispatcher) DispatchOnce(ctx context.Context, lk *lock.Lock) error {
	token := uuidx.New()
	ok, err := lk.Acquire(ctx, "outbox:dispatch", token, 5*time.Second)
	if err != nil || !ok {
		return err // another replica is dispatching
	}
	defer lk.Release(ctx, "outbox:dispatch", token) //nolint:errcheck

	var msgs []model.OutboxMessage
	q := d.db.WithContext(ctx)
	if q.Name() == "postgres" {
		// SKIP LOCKED is Postgres-only; other dialectors (sqlite tests)
		// rely on the distributed lock alone to serialise dispatch.
		q = q.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	}
	if err := q.Where("status = ?", StatusPending).
		Order("created_at").Limit(d.batch).Find(&msgs).Error; err != nil {
		return err
	}
	for i := range msgs {
		if err := d.publishWithRetry(ctx, &msgs[i]); err != nil {
			logger.Error("outbox message failed", zap.String("id", msgs[i].ID), zap.Error(err))
			d.markFailed(&msgs[i], err)
			continue
		}
		if err := d.db.Model(&msgs[i]).Updates(map[string]any{
			"status": StatusPublished, "published_at": time.Now(), "last_error": "",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) publishWithRetry(ctx context.Context, msg *model.OutboxMessage) error {
	body, _ := json.Marshal(msg)
	subject := d.subject(msg.EventType)
	return retry.Do(ctx, retry.Options{MaxAttempts: d.maxTry}, func(_ int) error {
		return d.breaker.Execute(func() error { return d.js.Publish(subject, body) })
	})
}

func (d *Dispatcher) markFailed(msg *model.OutboxMessage, cause error) {
	msg.Attempts++
	msg.LastError = cause.Error()
	status := StatusPending
	if msg.Attempts >= d.maxTry {
		status = StatusFailed
	}
	_ = d.db.Model(msg).Updates(map[string]any{
		"status": status, "attempts": msg.Attempts, "last_error": truncate(msg.LastError, 500),
	}).Error
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

