package worker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
)

func TestOpenSearchIndexerPostsBulk(t *testing.T) {
	var (
		gotPath string
		gotBody string
		gotAuth string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":false}`))
	}))
	defer srv.Close()

	idx := NewOpenSearchIndexer(srv.URL, "documents", "user", "pass")
	doc := &model.Document{
		BaseModel:      model.BaseModel{ID: "d1", UpdatedAt: time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)},
		DocumentNumber: "DOC-1", Title: "Annual Report", Status: "draft",
		OwnerID: "o1", Tags: []string{"finance"},
	}
	if err := idx.Index(context.Background(), doc); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/_bulk" {
		t.Fatalf("expected /_bulk, got %s", gotPath)
	}
	if gotAuth != "Basic dXNlcjpwYXNz" {
		t.Fatalf("unexpected auth %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"_index":"documents"`) || !strings.Contains(gotBody, `"_id":"d1"`) {
		t.Fatalf("bulk body missing index/id: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"title":"Annual Report"`) {
		t.Fatalf("bulk body missing document: %s", gotBody)
	}
}

func TestOpenSearchIndexerDeleteAndNotFound(t *testing.T) {
	var deleted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound) // tolerated
	}))
	defer srv.Close()

	idx := NewOpenSearchIndexer(srv.URL, "documents", "", "")
	if err := idx.Delete(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected a DELETE request")
	}
	// 404 on delete is not an error (document already gone).
	if err := idx.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("404 delete must be tolerated: %v", err)
	}
}

func TestOpenSearchIndexerToleratesTrailingSlash(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":false}`))
	}))
	defer srv.Close()

	idx := NewOpenSearchIndexer(srv.URL+"/", "documents", "", "")
	if err := idx.Index(context.Background(), &model.Document{BaseModel: model.BaseModel{ID: "d"}, Title: "t"}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/_bulk" {
		t.Fatalf("expected /_bulk (trailing slash trimmed), got %s", gotPath)
	}
}

func TestOpenSearchIndexerSurfacesServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	idx := NewOpenSearchIndexer(srv.URL, "documents", "", "")
	if err := idx.Index(context.Background(), &model.Document{BaseModel: model.BaseModel{ID: "x"}, Title: "t"}); err == nil {
		t.Fatal("expected an error on 503")
	}
}
