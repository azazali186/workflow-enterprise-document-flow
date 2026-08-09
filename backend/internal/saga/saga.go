// Package saga implements the saga pattern from the README: a saga's state
// lives in Redis (key saga:<id>) while each step transition publishes an
// outbox event so workers can react asynchronously.
package saga

import (
	"context"
	"errors"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"gorm.io/gorm"
)

// Status values.
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// StepStatus values.
const (
	StepPending   = "pending"
	StepRunning   = "running"
	StepCompleted = "completed"
)

// Step is one saga transition.
type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Saga is the persisted state of an orchestration.
type Saga struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	Status        string    `json:"status"`
	CurrentStep   int       `json:"current_step"`
	Steps         []Step    `json:"steps"`
	CreatedAt     time.Time `json:"created_at"`
}

// Key returns the Redis key for the saga.
func (s *Saga) Key() string { return "saga:" + s.ID }

// aggKey is the aggregate index key mapping an entity to its running saga.
func aggKey(aggregateType, aggregateID string) string {
	return "saga:agg:" + aggregateType + ":" + aggregateID
}

// Names of the README sagas.
const (
	DocumentUpload      = "document_upload"
	VerificationProcess = "verification_process"
	ApprovalChain       = "approval_chain"
	Archival            = "archival"
)

// DocumentUploadSteps mirrors the README upload flow.
var DocumentUploadSteps = []string{"upload", "virus_scan", "metadata_extraction", "storage", "indexing"}

// VerificationProcessSteps mirrors the README authenticity-validation flow.
var VerificationProcessSteps = []string{"authenticity_check", "document_update"}

// ApprovalChainSteps mirrors the README multi-level approval flow.
var ApprovalChainSteps = []string{"routing", "decision"}

// ArchivalSteps mirrors the README archival flow.
var ArchivalSteps = []string{"archive"}

// Coordinator manages saga lifecycle.
type Coordinator struct {
	cache *cache.Client
	db    *gorm.DB
}

// New wires the coordinator.
func New(c *cache.Client, db *gorm.DB) *Coordinator {
	return &Coordinator{cache: c, db: db}
}

// Start begins a saga with the given ordered steps.
func (c *Coordinator) Start(ctx context.Context, name, aggregateType, aggregateID string, stepNames []string) (*Saga, error) {
	steps := make([]Step, 0, len(stepNames))
	for _, n := range stepNames {
		steps = append(steps, Step{Name: n, Status: StepPending})
	}
	s := &Saga{
		ID:            uuidx.New(),
		Name:          name,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Status:        StatusRunning,
		CurrentStep:   0,
		Steps:         steps,
		CreatedAt:     time.Now(),
	}
	if len(steps) > 0 {
		s.Steps[0].Status = StepRunning
	}
	if err := c.cache.SetJSON(s.Key(), s, 24*time.Hour); err != nil {
		return nil, err
	}
	if err := c.cache.Set(aggKey(aggregateType, aggregateID), s.ID, 24*time.Hour); err != nil {
		return nil, err
	}
	_ = outbox.Enqueue(ctx, c.db, outbox.Event{
		AggregateType: aggregateType, AggregateID: aggregateID, EventType: "saga.started",
	}, map[string]any{"saga_id": s.ID, "name": name})
	return s, nil
}

// FindByAggregate locates the active saga for an entity via its index.
func (c *Coordinator) FindByAggregate(ctx context.Context, aggregateType, aggregateID string) (*Saga, error) {
	sagaID, err := c.cache.Get(aggKey(aggregateType, aggregateID))
	if err != nil || sagaID == "" {
		return nil, errors.New("saga not found for aggregate")
	}
	return c.Get(ctx, sagaID)
}

// CompleteStep marks a step done and advances the saga. Failed sagas are
// terminal: completing steps on them is a no-op.
func (c *Coordinator) CompleteStep(ctx context.Context, sagaID, stepName string) (*Saga, error) {
	s, err := c.Get(ctx, sagaID)
	if err != nil {
		return nil, err
	}
	if s.Status == StatusFailed {
		return s, nil
	}
	completed := 0
	for i := range s.Steps {
		if s.Steps[i].Name == stepName {
			if s.Steps[i].Status == StepCompleted {
				return s, nil // idempotent: step already done, no re-advance
			}
			s.Steps[i].Status = StepCompleted
		}
		if s.Steps[i].Status == StepCompleted {
			completed++
		}
	}
	if completed == len(s.Steps) {
		s.Status = StatusCompleted
	} else if s.CurrentStep < len(s.Steps)-1 {
		s.CurrentStep++
		if s.Steps[s.CurrentStep].Status != StepCompleted {
			s.Steps[s.CurrentStep].Status = StepRunning
		}
	}
	if err := c.cache.SetJSON(s.Key(), s, 24*time.Hour); err != nil {
		return nil, err
	}
	_ = outbox.Enqueue(ctx, c.db, outbox.Event{
		AggregateType: s.AggregateType, AggregateID: s.AggregateID, EventType: "saga." + stepName,
	}, map[string]any{"saga_id": s.ID, "step": stepName, "status": s.Status})
	return s, nil
}

// Fail marks the saga failed and records the reason.
func (c *Coordinator) Fail(ctx context.Context, sagaID, stepName, reason string) error {
	s, err := c.Get(ctx, sagaID)
	if err != nil {
		return err
	}
	s.Status = StatusFailed
	for i := range s.Steps {
		if s.Steps[i].Name == stepName {
			s.Steps[i].Status = StatusFailed
		}
	}
	if err := c.cache.SetJSON(s.Key(), s, 24*time.Hour); err != nil {
		return err
	}
	return outbox.Enqueue(ctx, c.db, outbox.Event{
		AggregateType: s.AggregateType, AggregateID: s.AggregateID, EventType: "saga.failed",
	}, map[string]any{"saga_id": s.ID, "step": stepName, "reason": reason})
}

// Get loads saga state from Redis.
func (c *Coordinator) Get(ctx context.Context, sagaID string) (*Saga, error) {
	var s Saga
	if err := c.cache.GetJSON("saga:"+sagaID, &s); err != nil {
		return nil, err
	}
	if s.ID == "" {
		return nil, errors.New("saga not found")
	}
	return &s, nil
}
