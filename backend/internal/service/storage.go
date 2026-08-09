package service

import (
	"context"
	"path/filepath"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
)

// StorageRegisterInput records a stored binary.
type StorageRegisterInput struct {
	DocumentID string
	Provider   string
	Bucket     string
	ObjectKey  string
	FileName   string
	MimeType   string
	SizeBytes  int64
	Checksum   string
}

// StorageService is the storage domain contract.
type StorageService interface {
	Register(ctx context.Context, actor Actor, in StorageRegisterInput) (*model.Storage, error)
	Get(ctx context.Context, id string) (*model.Storage, error)
	List(ctx context.Context, n *pagination.Normalized) ([]model.Storage, *pagination.Meta, map[string]any, error)
	Delete(ctx context.Context, actor Actor, id string) error
}

// storageService implements StorageService with field-level encryption.
type storageService struct {
	repo  *repository.Repo[model.Storage]
	docs  *repository.Repo[model.Document]
	audit *AuditService
}

// NewStorageService wires the storage domain.
func NewStorageService(repo *repository.Repo[model.Storage],
	docs *repository.Repo[model.Document], audit *AuditService) StorageService {
	return &storageService{repo: repo, docs: docs, audit: audit}
}

func (s *storageService) Register(ctx context.Context, actor Actor, in StorageRegisterInput) (*model.Storage, error) {
	if _, err := s.docs.GetByID(ctx, in.DocumentID); err != nil {
		return nil, err
	}
	// Object keys are stored encrypted and later resolved against the object
	// store (S3 bucket or local directory), so a client-supplied key must be
	// a safe relative path: no absolute paths, no ".." elements, no NUL. This
	// blocks path traversal on the local store and key injection on S3.
	if in.ObjectKey != "" && !filepath.IsLocal(in.ObjectKey) {
		return nil, apperror.BadRequest("object_key must be a safe relative path")
	}
	if in.Provider == "" {
		in.Provider = "local"
	}
	encKey, err := crypto.Encrypt(in.ObjectKey) // sensitive at rest
	if err != nil {
		return nil, err
	}
	rec := &model.Storage{
		DocumentID: in.DocumentID,
		Provider:   in.Provider,
		Bucket:     in.Bucket,
		ObjectKey:  encKey,
		FileName:   in.FileName,
		MimeType:   in.MimeType,
		SizeBytes:  in.SizeBytes,
		Checksum:   in.Checksum,
		Status:     "stored",
		StoredAt:   nowPtr(),
	}
	if err := s.repo.Create(ctx, rec); err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, actor, "create", "storage", rec.ID, nil, rec)
	return rec, nil
}

func (s *storageService) Get(ctx context.Context, id string) (*model.Storage, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *storageService) List(ctx context.Context, n *pagination.Normalized) ([]model.Storage, *pagination.Meta, map[string]any, error) {
	return s.repo.List(ctx, repository.ListQuery{P: n})
}

func (s *storageService) Delete(ctx context.Context, actor Actor, id string) error {
	before, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.audit.Record(ctx, actor, "delete", "storage", id, before, nil)
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}
