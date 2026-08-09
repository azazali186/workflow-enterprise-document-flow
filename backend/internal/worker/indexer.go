package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
)

// NoopIndexer records nothing; documents remain searchable only via the DB
// list endpoints.
type NoopIndexer struct{}

// Index implements service.Indexer.
func (NoopIndexer) Index(context.Context, *model.Document) error { return nil }

// Delete implements service.Indexer.
func (NoopIndexer) Delete(context.Context, string) error { return nil }

// OpenSearchIndexer writes documents to an OpenSearch/Elasticsearch cluster
// through the _bulk API.
type OpenSearchIndexer struct {
	url      string
	index    string
	username string
	password string
	client   *http.Client
}

// NewOpenSearchIndexer wires the bulk indexer. A trailing slash on url is
// tolerated so dashboard-pasted endpoints don't produce "//_bulk" paths.
func NewOpenSearchIndexer(url, index, username, password string) *OpenSearchIndexer {
	return &OpenSearchIndexer{
		url:      strings.TrimRight(url, "/"),
		index:    index,
		username: username,
		password: password,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// doc is the searchable projection of a document.
type doc struct {
	ID             string   `json:"id"`
	DocumentNumber string   `json:"document_number"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         string   `json:"status"`
	OwnerID        string   `json:"owner_id"`
	CategoryID     string   `json:"category_id,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	UpdatedAt      string   `json:"updated_at"`
}

func (i *OpenSearchIndexer) projection(d *model.Document) doc {
	var tags []string
	if d.Tags != nil {
		tags = d.Tags
	}
	var categoryID string
	if d.CategoryID != nil {
		categoryID = *d.CategoryID
	}
	return doc{
		ID: d.ID, DocumentNumber: d.DocumentNumber, Title: d.Title,
		Description: d.Description, Status: d.Status, OwnerID: d.OwnerID,
		CategoryID: categoryID, Tags: tags,
		UpdatedAt: d.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// Index implements Indexer with an NDJSON _bulk request.
func (i *OpenSearchIndexer) Index(ctx context.Context, d *model.Document) error {
	if d == nil || d.ID == "" {
		return nil
	}
	action, _ := json.Marshal(map[string]any{
		"index": map[string]any{"_index": i.index, "_id": d.ID},
	})
	body, err := json.Marshal(i.projection(d))
	if err != nil {
		return err
	}
	payload := bytes.Join([][]byte{action, body}, []byte("\n"))
	payload = append(payload, '\n')

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.url+"/_bulk", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if i.username != "" {
		req.SetBasicAuth(i.username, i.password)
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("opensearch bulk failed: %d %s", resp.StatusCode, string(raw))
	}
	return nil
}

// Delete implements Indexer.
func (i *OpenSearchIndexer) Delete(ctx context.Context, docID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/%s/_doc/%s", i.url, i.index, docID), nil)
	if err != nil {
		return err
	}
	if i.username != "" {
		req.SetBasicAuth(i.username, i.password)
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("opensearch delete failed: %d", resp.StatusCode)
	}
	return nil
}
