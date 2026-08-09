package service

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"gorm.io/gorm"
)

// SearchService is the README search module (SearchService): keyword search
// over documents with the same whitelisted filters, dynamic sort and cursor
// pagination as the list endpoints. Searches run against Postgres (full-text
// LIKE across title, document number and description); when
// INDEXER=opensearch is configured the external index is kept in sync by the
// pipeline, ready for direct-index queries.
type SearchService interface {
	SearchDocuments(ctx context.Context, userID string, n *pagination.Normalized) ([]model.Document, *pagination.Meta, error)
}

// searchService implements SearchService on top of the documents repository.
type searchService struct {
	db   *gorm.DB
	docs *repository.Repo[model.Document]
}

// NewSearchService wires the search domain.
func NewSearchService(db *gorm.DB, docs *repository.Repo[model.Document]) SearchService {
	return &searchService{db: db, docs: docs}
}

// SearchDocuments runs the keyword search. A non-empty search term is
// required; everything else (filters, sort, date range, cursor) is carried by
// the normalized request. Results are restricted to documents the caller may
// read (same row-level access scope as the documents list).
func (s *searchService) SearchDocuments(ctx context.Context, userID string, n *pagination.Normalized) ([]model.Document, *pagination.Meta, error) {
	if n == nil || n.Search == "" {
		return nil, nil, apperror.BadRequest("search term is required")
	}
	items, meta, _, err := s.docs.List(ctx, repository.ListQuery{
		P:      n,
		Scopes: []func(*gorm.DB) *gorm.DB{documentAccessScope(ctx, s.db, userID)},
	})
	if err != nil {
		return nil, nil, err
	}
	return items, meta, nil
}

var _ SearchService = (*searchService)(nil)
