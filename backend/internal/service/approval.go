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

// CreateApprovalInput opens a multi-level approval chain for a document.
type CreateApprovalInput struct {
	DocumentID  string
	ApproverIDs []string
}

// DecideApprovalInput records one approver decision.
type DecideApprovalInput struct {
	ApprovalID string
	Decision   string // approved | rejected
	Comment    string
}

// ApprovalService is the approval domain contract.
type ApprovalService interface {
	CreateChain(ctx context.Context, actor Actor, in CreateApprovalInput) ([]model.Approval, error)
	Decide(ctx context.Context, actor Actor, in DecideApprovalInput) (*model.Approval, error)
	Get(ctx context.Context, id string) (*model.Approval, error)
	List(ctx context.Context, n *pagination.Normalized) ([]model.Approval, *pagination.Meta, map[string]any, error)
}

// approvalService implements ApprovalService.
type approvalService struct {
	db    *gorm.DB
	repo  *repository.Repo[model.Approval]
	docs  *repository.Repo[model.Document]
	audit *AuditService
	locks *lock.Lock
	sagas *saga.Coordinator
}

// NewApprovalService wires the approval domain.
func NewApprovalService(db *gorm.DB, repo *repository.Repo[model.Approval],
	docs *repository.Repo[model.Document], audit *AuditService, lk *lock.Lock,
	sagas *saga.Coordinator) ApprovalService {
	return &approvalService{db: db, repo: repo, docs: docs, audit: audit, locks: lk, sagas: sagas}
}

func (s *approvalService) CreateChain(ctx context.Context, actor Actor, in CreateApprovalInput) ([]model.Approval, error) {
	if _, err := s.docs.GetByID(ctx, in.DocumentID); err != nil {
		return nil, err
	}
	if len(in.ApproverIDs) == 0 {
		return nil, apperror.BadRequest("at least one approver is required")
	}
	chain := make([]model.Approval, 0, len(in.ApproverIDs))
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, approverID := range in.ApproverIDs {
			a := model.Approval{
				DocumentID:  in.DocumentID,
				Level:       i + 1,
				ApproverID:  approverID,
				RequestedBy: actor.ID,
				Status:      constant.StatusPending,
			}
			if err := tx.Create(&a).Error; err != nil {
				return err
			}
			chain = append(chain, a)
		}
		return outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "approval", AggregateID: in.DocumentID, EventType: constant.EventApprovalRequired,
		}, map[string]any{"document_id": in.DocumentID, "levels": len(in.ApproverIDs)})
	})
	if err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, actor, constant.ActionCreate, "approval", in.DocumentID, nil, chain)
	// Kick off the README ApprovalChain saga, then mark the chain routed to
	// its approvers (the decision step completes when every level is decided).
	if sts, err := s.sagas.Start(ctx, saga.ApprovalChain, "approval", in.DocumentID, saga.ApprovalChainSteps); err == nil {
		_, _ = s.sagas.CompleteStep(ctx, sts.ID, "routing")
	}
	return chain, nil
}

func (s *approvalService) Decide(ctx context.Context, actor Actor, in DecideApprovalInput) (*model.Approval, error) {
	if in.Decision != constant.StatusApproved && in.Decision != constant.StatusRejected {
		return nil, apperror.BadRequest("decision must be approved or rejected")
	}
	token := uuidx.New()
	var docID string
	resolved := false
	err := s.locks.WithLock(ctx, "approval:"+in.ApprovalID, token, 10*time.Second, func() error {
		a, err := s.repo.GetByID(ctx, in.ApprovalID)
		if err != nil {
			return err
		}
		if a.ApproverID != actor.ID {
			return apperror.Forbidden("not the assigned approver")
		}
		if a.Status != constant.StatusPending {
			return apperror.Conflict("approval already decided")
		}
		now := time.Now()
		a.Status = in.Decision
		a.Comment = in.Comment
		a.DecidedAt = &now
		if err := s.db.WithContext(ctx).Save(a).Error; err != nil {
			return err
		}
		docID = a.DocumentID
		_ = s.audit.Change(ctx, actor, "approval", a.ID, nil, a)
		resolved, err = s.maybeResolveDocument(ctx, a, in.Decision)
		return err
	})
	if err != nil {
		return nil, err
	}
	// Only when this decision closed the whole chain is the ApprovalChain
	// saga's decision step done (best-effort, like verification).
	if resolved {
		if sts, err := s.sagas.FindByAggregate(ctx, "approval", docID); err == nil {
			_, _ = s.sagas.CompleteStep(ctx, sts.ID, "decision")
		}
	}
	return s.repo.GetByID(ctx, in.ApprovalID)
}

// maybeResolveDocument closes the chain when every level is decided and
// reports whether the chain was fully resolved by this decision.
func (s *approvalService) maybeResolveDocument(ctx context.Context, a *model.Approval, decision string) (bool, error) {
	var open int64
	if err := s.db.WithContext(ctx).Model(&model.Approval{}).
		Where("document_id = ? AND status = ?", a.DocumentID, constant.StatusPending).Count(&open).Error; err != nil {
		return false, err
	}
	if open > 0 {
		return false, nil // chain continues
	}
	docStatus := constant.DocApproved
	eventType := constant.EventDocumentReady
	if decision == constant.StatusRejected {
		docStatus = constant.DocRejected
		eventType = "approval_rejected"
	}
	if err := s.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", a.DocumentID).Update("status", docStatus).Error; err != nil {
		return false, err
	}
	err := outbox.Enqueue(ctx, s.db, outbox.Event{
		AggregateType: "approval", AggregateID: a.DocumentID, EventType: eventType,
	}, map[string]any{"document_id": a.DocumentID, "status": docStatus})
	return err == nil, err
}

func (s *approvalService) Get(ctx context.Context, id string) (*model.Approval, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *approvalService) List(ctx context.Context, n *pagination.Normalized) ([]model.Approval, *pagination.Meta, map[string]any, error) {
	return s.repo.List(ctx, repository.ListQuery{P: n})
}

var _ ApprovalService = (*approvalService)(nil)
