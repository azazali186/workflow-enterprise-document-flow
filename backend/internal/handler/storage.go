package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/aeroxe/docu-flow/backend/internal/objectstore"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// StorageHandler exposes storage endpoints backed by an ObjectStore.
type StorageHandler struct {
	svc   service.StorageService
	store objectstore.ObjectStore
}

// NewStorageHandler wires the handler with an object store.
func NewStorageHandler(svc service.StorageService, store objectstore.ObjectStore) *StorageHandler {
	return &StorageHandler{svc: svc, store: store}
}

// Upload handles POST /api/v1/storages/upload (multipart).
// @Summary      Upload a document binary (multipart form)
// @Tags         storages
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        document_id formData string true "Document id"
// @Param        file formData file true "File to store"
// @Success      200 {object} response.Response
// @Router       /api/v1/storages/upload [post]
func (h *StorageHandler) Upload(ctx context.Context, c *app.RequestContext) {
	docID := string(c.FormValue("document_id"))
	if docID == "" {
		writeError(c, apperror.BadRequest("document_id is required"))
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		writeError(c, apperror.BadRequest("file is required"))
		return
	}
	key := objectstore.NewKey(file.Filename)
	src, err := file.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer func() { _ = src.Close() }()
	// Stream once while computing the SHA-256 checksum (no full read into RAM).
	hash := sha256.New()
	n, putErr := h.store.Put(ctx, key, io.TeeReader(src, hash), file.Size, file.Header.Get("Content-Type"))
	if putErr != nil {
		writeError(c, apperror.Internal("failed to persist upload", putErr))
		return
	}
	rec, err := h.svc.Register(ctx, actor(c), service.StorageRegisterInput{
		DocumentID: docID,
		Provider:   h.store.Provider(),
		Bucket:     h.store.Bucket(),
		ObjectKey:  key,
		FileName:   file.Filename,
		MimeType:   file.Header.Get("Content-Type"),
		SizeBytes:  n,
		Checksum:   hex.EncodeToString(hash.Sum(nil)),
	})
	if err != nil {
		_ = h.store.Delete(ctx, key)
		writeError(c, err)
		return
	}
	response.Success(rec).Json(c)
}

type storageRegisterRequest struct {
	DocumentID string `json:"document_id" validate:"required"`
	Provider   string `json:"provider,omitempty"`
	Bucket     string `json:"bucket,omitempty"`
	ObjectKey  string `json:"object_key,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
}

// Register handles POST /api/v1/storages/register.
// @Summary      Record storage metadata for an already-persisted binary
// @Tags         storages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body storageRegisterRequest true "Storage metadata"
// @Success      200 {object} response.Response
// @Router       /api/v1/storages/register [post]
func (h *StorageHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req storageRegisterRequest
	if !bind(c, &req) {
		return
	}
	rec, err := h.svc.Register(ctx, actor(c), service.StorageRegisterInput{
		DocumentID: req.DocumentID, Provider: req.Provider, Bucket: req.Bucket,
		ObjectKey: req.ObjectKey, FileName: req.FileName, MimeType: req.MimeType,
		SizeBytes: req.SizeBytes, Checksum: req.Checksum,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(rec).Json(c)
}

// Get handles POST /api/v1/storages/get.
// @Summary      Get a storage record by id
// @Tags         storages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Storage id"
// @Success      200 {object} response.Response
// @Router       /api/v1/storages/get [post]
func (h *StorageHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	rec, err := h.svc.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(rec).Json(c)
}

// Delete handles POST /api/v1/storages/delete.
// @Summary      Soft-delete a storage record
// @Tags         storages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Storage id"
// @Success      200 {object} response.Response
// @Router       /api/v1/storages/delete [post]
func (h *StorageHandler) Delete(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.Delete(ctx, actor(c), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(nil).Json(c)
}

// List handles POST /api/v1/storages/list.
// @Summary      List storage records with cursor pagination
// @Tags         storages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/storages/list [post]
func (h *StorageHandler) List(ctx context.Context, c *app.RequestContext) {
	n, ok := normalize(c, "created_at")
	if !ok {
		return
	}
	items, meta, summary, err := h.svc.List(ctx, n)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(response.Page{Items: items, Pagination: meta, Summary: summary}).Json(c)
}
