package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// DocumentHandler exposes document endpoints.
type DocumentHandler struct {
	svc service.DocumentService
}

// NewDocumentHandler wires the handler.
func NewDocumentHandler(svc service.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

type documentCreateRequest struct {
	Title       string         `json:"title" validate:"required"`
	Description string         `json:"description,omitempty"`
	CategoryID  string         `json:"category_id,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Meta        model.JSONMap  `json:"meta,omitempty"`
}

// Create handles POST /api/v1/documents/create.
// @Summary      Register a document (starts the upload saga)
// @Tags         documents
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body documentCreateRequest true "Document payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/documents/create [post]
func (h *DocumentHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req documentCreateRequest
	if !bind(c, &req) {
		return
	}
	doc, err := h.svc.Create(ctx, actor(c), service.CreateDocumentInput{
		Title: req.Title, Description: req.Description, CategoryID: req.CategoryID,
		Tags: req.Tags, Meta: req.Meta,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(doc).Json(c)
}

type documentUpdateRequest struct {
	ID          string        `json:"id" validate:"required"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	CategoryID  string        `json:"category_id,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Meta        model.JSONMap `json:"meta,omitempty"`
	Status      string        `json:"status,omitempty"`
}

// Update handles PATCH /api/v1/documents/update.
// @Summary      Update a document (creates a new version snapshot)
// @Tags         documents
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body documentUpdateRequest true "Document update payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/documents/update [patch]
func (h *DocumentHandler) Update(ctx context.Context, c *app.RequestContext) {
	var req documentUpdateRequest
	if !bind(c, &req) {
		return
	}
	doc, err := h.svc.Update(ctx, actor(c), service.UpdateDocumentInput{
		ID: req.ID, Title: req.Title, Description: req.Description, CategoryID: req.CategoryID,
		Tags: req.Tags, Meta: req.Meta, Status: req.Status,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(doc).Json(c)
}

// Delete handles POST /api/v1/documents/delete.
// @Summary      Soft-delete a document
// @Tags         documents
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Document id"
// @Success      200 {object} response.Response
// @Router       /api/v1/documents/delete [post]
func (h *DocumentHandler) Delete(ctx context.Context, c *app.RequestContext) {
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

// Get handles POST /api/v1/documents/get.
// @Summary      Get a document by id (cache backed)
// @Tags         documents
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Document id"
// @Success      200 {object} response.Response
// @Router       /api/v1/documents/get [post]
func (h *DocumentHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	doc, err := h.svc.Get(ctx, actor(c), req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(doc).Json(c)
}

// List handles POST /api/v1/documents/list.
// @Summary      List documents with cursor pagination, filters, sort and summary
// @Tags         documents
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/documents/list [post]
func (h *DocumentHandler) List(ctx context.Context, c *app.RequestContext) {
	n, ok := normalize(c, "created_at")
	if !ok {
		return
	}
	items, meta, summary, err := h.svc.List(ctx, actor(c), n)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(response.Page{Items: items, Pagination: meta, Summary: summary}).Json(c)
}
