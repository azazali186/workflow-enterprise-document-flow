package service

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
)

// Indexer makes documents searchable in an external search cluster. The
// worker implements it (noop + OpenSearch); the document service depends on
// it so deletions are removed from the index synchronously.
type Indexer interface {
	Index(ctx context.Context, doc *model.Document) error
	Delete(ctx context.Context, docID string) error
}
