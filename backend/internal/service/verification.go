package service

import (
	"context"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/lock"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"gorm.io/gorm"
)

// CreateVerificationInput starts a verification request.
type CreateVerificationInput struct {
	DocumentID string
	Method     string
	Notes      string
}

// DecideVerificationInput records a verification outcome.
type DecideVerificationInput struct {
	VerificationID string
	Decision       string // verified | rejected
	Notes          string
}

// VerificationService is the verification domain contract.
type VerificationService interface {
	Create(ctx context.Context, actor Actor, in CreateVerificationInput) (*model.Verification, error)
	Decide(ctx context.Context, actor Actor, in DecideVerificationInput) (*model.Verification, error)
	Get(ctx context.Context, id string) (*model.Verification, error)
	List(ctx context.Context, n *pagination.Normalized) ([]model.Verification, *pagination.Meta, map[string]any, error)
}

// verificationService implements VerificationService.
type verificationService struct {
	db    *gorm.DB
	repo  *repository.Repo[model.Verification]
	docs  *repository.Repo[model.Document]
	audit *AuditService
	locks *lock.Lock
	sagas *saga.Coordinator
}

// NewVerificationService wires the verification domain.
func NewVerificationService(db *gorm.DB, repo *repository.Repo[model.Verification],
	docs *repository.Repo[model.Document], audit *AuditService, lk *lock.Lock,
	sagas *saga.Coordinator) VerificationService {
	return &verificationService{db: db, repo: repo, docs: docs, audit: audit, locks: lk, sagas: sagas}
}

func (s *verificationService) Create(ctx context.Context, actor Actor, in CreateVerificationInput) (*model.Verification, error) {
	if _, err := s.docs.GetByID(ctx, in.DocumentID); err != nil {
		return nil, err
	}
	if in.Method == "" {
		in.Method = "manual"
	}
	ver := &model.Verification{
		DocumentID:  in.DocumentID,
		RequestedBy: actor.ID,
		Status:      constant.StatusPending,
		Method:      in.Method,
		Notes:       in.Notes,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ver).Error; err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "verification", AggregateID: ver.ID, EventType: constant.EventVerificationNeeded,
		}, map[string]any{"document_id": in.DocumentID, "verification_id": ver.ID})
	})
	if err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, actor, constant.ActionCreate, "verification", ver.ID, nil, ver)
	// Kick off the README VerificationProcess saga (state in Redis, steps
	// advanced asynchronously via NATS once the decision lands).
	_, _ = s.sagas.Start(ctx, saga.VerificationProcess, "verification", ver.ID, saga.VerificationProcessSteps)
	return ver, nil
}

func (s *verificationService) Decide(ctx context.Context, actor Actor, in DecideVerificationInput) (*model.Verification, error) {
	if in.Decision != constant.StatusVerified && in.Decision != constant.StatusRejected {
		return nil, apperror.BadRequest("decision must be verified or rejected")
	}
	// Distributed lock serialises concurrent decisions on the same record.
	token := uuidx.New()
	err := s.locks.WithLock(ctx, "verification:"+in.VerificationID, token, 10*time.Second, func() error {
		ver, err := s.repo.GetByID(ctx, in.VerificationID)
		if err != nil {
			return err
		}
		if ver.Status != constant.StatusPending {
			return apperror.Conflict("verification already decided")
		}
		now := time.Now()
		ver.Status = in.Decision
		ver.VerifiedBy = &actor.ID
		ver.Notes = in.Notes
		ver.VerifiedAt = &now
		if err := s.db.WithContext(ctx).Save(ver).Error; err != nil {
			return err
		}
		// Cascade to the document lifecycle.
		newStatus := constant.DocVerified
		eventType := constant.EventDocumentReady
		if in.Decision == constant.StatusRejected {
			newStatus = constant.DocRejected
			eventType = "verification_rejected"
		}
		if err := s.db.WithContext(ctx).Model(&model.Document{}).
			Where("id = ?", ver.DocumentID).Update("status", newStatus).Error; err != nil {
			return err
		}
		_ = s.audit.Change(ctx, actor, "verification", ver.ID, nil, ver)
		return outbox.Enqueue(ctx, s.db, outbox.Event{
			AggregateType: "verification", AggregateID: ver.ID, EventType: eventType,
		}, map[string]any{"document_id": ver.DocumentID, "status": newStatus})
	})
	if err != nil {
		return nil, err
	}
	// The decision IS the verification work: complete the saga synchronously
	// (best-effort — a decision on a legacy record without a saga must not
	// fail the request).
	s.completeVerificationSaga(ctx, in.VerificationID)
	return s.repo.GetByID(ctx, in.VerificationID)
}

// completeVerificationSaga advances the VerificationProcess saga through its
// remaining steps once a decision has been recorded. Best-effort.
func (s *verificationService) completeVerificationSaga(ctx context.Context, verificationID string) {
	sts, err := s.sagas.FindByAggregate(ctx, "verification", verificationID)
	if err != nil {
		return // no active saga (legacy row or saga not started)
	}
	for _, step := range saga.VerificationProcessSteps {
		if _, err := s.sagas.CompleteStep(ctx, sts.ID, step); err != nil {
			return // idempotent on the next call; stop advancing on failure
		}
	}
}

func (s *verificationService) Get(ctx context.Context, id string) (*model.Verification, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *verificationService) List(ctx context.Context, n *pagination.Normalized) ([]model.Verification, *pagination.Meta, map[string]any, error) {
	return s.repo.List(ctx, repository.ListQuery{P: n})
}

var _ VerificationService = (*verificationService)(nil)
