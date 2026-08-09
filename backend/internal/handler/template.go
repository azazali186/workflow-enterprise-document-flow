package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// TemplateHandler exposes template endpoints.
type TemplateHandler struct {
	crud *service.CrudService[model.Template]
}

// NewTemplateHandler wires the handler.
func NewTemplateHandler(crud *service.CrudService[model.Template]) *TemplateHandler {
	return &TemplateHandler{crud: crud}
}

type templateCreateRequest struct {
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug" validate:"required"`
	Description string `json:"description,omitempty"`
	CategoryID  string `json:"category_id,omitempty"`
	Content     string `json:"content,omitempty"`
}

// Create handles POST /api/v1/templates/create.
// @Summary      Create a template
// @Tags         templates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body templateCreateRequest true "Template payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/templates/create [post]
func (h *TemplateHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req templateCreateRequest
	if !bind(c, &req) {
		return
	}
	t := &model.Template{Name: req.Name, Slug: req.Slug, Description: req.Description,
		CategoryID: model.NullableString(req.CategoryID), Content: req.Content, Version: 1, IsActive: true,
		CreatedBy: userID(c)}
	t, err := h.crud.Create(ctx, actor(c), t)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(t).Json(c)
}

type templateUpdateRequest struct {
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	CategoryID  string `json:"category_id,omitempty"`
	Content     string `json:"content,omitempty"`
	Version     *int   `json:"version,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// Update handles PATCH /api/v1/templates/update.
// @Summary      Update a template
// @Tags         templates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body templateUpdateRequest true "Template update payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/templates/update [patch]
func (h *TemplateHandler) Update(ctx context.Context, c *app.RequestContext) {
	var req templateUpdateRequest
	if !bind(c, &req) {
		return
	}
	t := &model.Template{BaseModel: newBaseModel(req.ID), Name: req.Name,
		Description: req.Description, Content: req.Content}
	if req.CategoryID != "" {
		t.CategoryID = model.NullableString(req.CategoryID)
	}
	if req.Version != nil {
		t.Version = *req.Version
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}
	t, err := h.crud.Update(ctx, actor(c), t)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(t).Json(c)
}

// Delete handles POST /api/v1/templates/delete.
// @Summary      Soft-delete a template
// @Tags         templates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Template id"
// @Success      200 {object} response.Response
// @Router       /api/v1/templates/delete [post]
func (h *TemplateHandler) Delete(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	if err := h.crud.Delete(ctx, actor(c), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(nil).Json(c)
}

// Get handles POST /api/v1/templates/get.
// @Summary      Get a template by id
// @Tags         templates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Template id"
// @Success      200 {object} response.Response
// @Router       /api/v1/templates/get [post]
func (h *TemplateHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.crud.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(t).Json(c)
}

// List handles POST /api/v1/templates/list.
// @Summary      List templates with cursor pagination
// @Tags         templates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/templates/list [post]
func (h *TemplateHandler) List(ctx context.Context, c *app.RequestContext) {
	n, ok := normalize(c, "created_at")
	if !ok {
		return
	}
	items, meta, summary, err := h.crud.List(ctx, n)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(response.Page{Items: items, Pagination: meta, Summary: summary}).Json(c)
}
