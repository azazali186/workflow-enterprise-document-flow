package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"gorm.io/gorm"
)

// CreateDocumentInput carries document registration payload.
type CreateDocumentInput struct {
	Title       string
	Description string
	CategoryID  string
	Tags        []string
	Meta        model.JSONMap
	OwnerID     string
}

// UpdateDocumentInput carries document update payload.
type UpdateDocumentInput struct {
	ID          string
	Title       string
	Description string
	CategoryID  string
	Tags        []string
	Meta        model.JSONMap
	Status      string
}

// DocumentService is the document domain contract.
//
// Row-level access control is enforced on every read and write: callers may
// read documents they own or hold an active grant for (super admins see
// everything) and may modify documents they own or hold a write/approve
// grant for. Denials surface as a not-found error so existence is not leaked.
type DocumentService interface {
	Create(ctx context.Context, actor Actor, in CreateDocumentInput) (*model.Document, error)
	Update(ctx context.Context, actor Actor, in UpdateDocumentInput) (*model.Document, error)
	Delete(ctx context.Context, actor Actor, id string) error
	Get(ctx context.Context, actor Actor, id string) (*model.Document, error)
	List(ctx context.Context, actor Actor, n *pagination.Normalized) ([]model.Document, *pagination.Meta, map[string]any, error)
}

// DocCache is the narrow cache interface the document service depends on.
type DocCache interface {
	SetJSON(key string, v any, ttl time.Duration) error
	GetJSON(key string, out any) error
	Del(keys ...string) error
}

// documentService implements DocumentService.
type documentService struct {
	db       *gorm.DB
	repo     *repository.Repo[model.Document]
	versions *repository.Repo[model.Version]
	audit    *AuditService
	cache    DocCache
	sagas    *saga.Coordinator
	indexer  Indexer
}

// NewDocumentService wires the document domain.
func NewDocumentService(db *gorm.DB, repo *repository.Repo[model.Document],
	versions *repository.Repo[model.Version], audit *AuditService, c DocCache,
	sagas *saga.Coordinator, indexer Indexer) DocumentService {
	return &documentService{db: db, repo: repo, versions: versions, audit: audit,
		cache: c, sagas: sagas, indexer: indexer}
}

func (s *documentService) Create(ctx context.Context, actor Actor, in CreateDocumentInput) (*model.Document, error) {
	if in.Title == "" {
		return nil, apperror.BadRequest("title is required")
	}
	ownerID := in.OwnerID
	if ownerID == "" {
		ownerID = actor.ID
	}
	doc := &model.Document{
		DocumentNumber: nextDocumentNumber(),
		Title:          in.Title,
		Description:    in.Description,
		CategoryID:     model.NullableString(in.CategoryID),
		OwnerID:        ownerID,
		Status:         constant.DocDraft,
		Meta:           in.Meta,
		Tags:           in.Tags,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "document", AggregateID: doc.ID, EventType: constant.EventDocumentUploaded,
		}, map[string]any{"document_id": doc.ID, "document_number": doc.DocumentNumber, "title": doc.Title})
	})
	if err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, actor, constant.ActionCreate, "document", doc.ID, nil, doc)
	_, _ = s.sagas.Start(ctx, saga.DocumentUpload, "document", doc.ID, saga.DocumentUploadSteps)
	return doc, nil
}

func (s *documentService) Update(ctx context.Context, actor Actor, in UpdateDocumentInput) (*model.Document, error) {
	before, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if ok, err := documentModifiable(ctx, s.db, actor.ID, before); err != nil {
		return nil, err
	} else if !ok {
		return nil, apperror.NotFound("document")
	}
	beforeSnapshot := *before // pre-update state for the audit trail
	if in.Title != "" {
		before.Title = in.Title
	}
	if in.Description != "" {
		before.Description = in.Description
	}
	if in.CategoryID != "" {
		before.CategoryID = model.NullableString(in.CategoryID)
	}
	if in.Tags != nil {
		before.Tags = in.Tags
	}
	if in.Meta != nil {
		before.Meta = in.Meta
	}
	if in.Status != "" && validStatusTransition(before.Status, in.Status) {
		before.Status = in.Status
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxVer int64
		_ = tx.Model(&model.Version{}).Where("document_id = ?", in.ID).Count(&maxVer)
		snapshot := toJSONMap(before)
		version := &model.Version{
			DocumentID:    in.ID,
			VersionNumber: int(maxVer) + 1,
			ChangeSummary: "metadata updated",
			Snapshot:      snapshot,
			CreatedBy:     actor.ID,
		}
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		if err := tx.Save(before).Error; err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "document", AggregateID: in.ID, EventType: constant.EventDocumentUpdated,
		}, map[string]any{"document_id": in.ID, "version": version.VersionNumber})
	})
	if err != nil {
		return nil, err
	}
	_ = s.cache.Del("cache:document:" + in.ID)
	_ = s.audit.Change(ctx, actor, "document", in.ID, &beforeSnapshot, before)
	// Keep the search index fresh: updates change exactly the fields a search
	// index displays (title, status, tags). Best-effort; noop unless configured.
	if s.indexer != nil {
		_ = s.indexer.Index(ctx, before)
	}
	// A transition to archived starts the README Archival saga. Archiving is
	// synchronous here (the status already changed), so the single step is
	// completed immediately — the saga lifecycle is still tracked in Redis
	// and emitted over NATS for consumers. A distinct aggregate type keeps
	// the saga index separate from the document_upload saga, so an early
	// archive can never hijack the worker's lookup of a running upload saga.
	if in.Status == constant.DocArchived && validStatusTransition(beforeSnapshot.Status, in.Status) {
		if sts, err := s.sagas.Start(ctx, saga.Archival, "document-archive", in.ID, saga.ArchivalSteps); err == nil {
			_, _ = s.sagas.CompleteStep(ctx, sts.ID, "archive")
		}
	}
	return before, nil
}

func (s *documentService) Delete(ctx context.Context, actor Actor, id string) error {
	before, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ok, err := documentModifiable(ctx, s.db, actor.ID, before); err != nil {
		return err
	} else if !ok {
		return apperror.NotFound("document")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.cache.Del("cache:document:" + id)
	// Keep the external search index consistent (best-effort; the indexer is
	// a noop unless INDEXER=opensearch is configured).
	if s.indexer != nil {
		_ = s.indexer.Delete(ctx, id)
	}
	return s.audit.Record(ctx, actor, constant.ActionDelete, "document", id, before, nil)
}

func (s *documentService) Get(ctx context.Context, actor Actor, id string) (*model.Document, error) {
	// Access is checked before the shared cache is consulted: the cache is
	// keyed by document id only, so a cached row must never be served to a
	// caller without a grant. The check resolves ownership from the DB, so
	// owners and super admins still enjoy the cache fast path.
	if ok, err := documentReadableID(ctx, s.db, actor.ID, id); err != nil {
		return nil, err
	} else if !ok {
		// Not an owner, not a super admin, and no active grant: existence is
		// not revealed — answer exactly like a missing row.
		return nil, apperror.NotFound("document")
	}
	var cached model.Document
	if err := s.cache.GetJSON("cache:document:"+id, &cached); err == nil && cached.ID != "" {
		return &cached, nil
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetJSON("cache:document:"+id, doc, constant.EntityCacheTTL)
	return doc, nil
}

func (s *documentService) List(ctx context.Context, actor Actor, n *pagination.Normalized) ([]model.Document, *pagination.Meta, map[string]any, error) {
	return s.repo.List(ctx, repository.ListQuery{
		P:      n,
		Scopes: []func(*gorm.DB) *gorm.DB{documentAccessScope(ctx, s.db, actor.ID)},
	})
}

func validStatusTransition(from, to string) bool {
	allowed := map[string]map[string]bool{
		constant.DocDraft:        {constant.DocPendingVerif: true, constant.DocArchived: true},
		constant.DocPendingVerif: {constant.DocVerified: true, constant.DocRejected: true},
		constant.DocVerified:     {constant.DocApproved: true, constant.DocArchived: true},
		constant.DocRejected:     {constant.DocDraft: true},
		constant.DocApproved:     {constant.DocArchived: true},
	}
	if m, ok := allowed[from]; ok {
		return m[to]
	}
	return false
}

// toJSONMap converts a struct to a plain map for version snapshots.
func toJSONMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return m
}

func nextDocumentNumber() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec // G404: human-facing doc number, not a secret; rand is auto-seeded since Go 1.20
	}
	return fmt.Sprintf("DOC-%s-%s", time.Now().Format("20060102"), string(b))
}
