package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// CategoryHandler exposes category endpoints.
type CategoryHandler struct {
	crud *service.CrudService[model.Category]
}

// NewCategoryHandler wires the handler.
func NewCategoryHandler(crud *service.CrudService[model.Category]) *CategoryHandler {
	return &CategoryHandler{crud: crud}
}

type categoryCreateRequest struct {
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug" validate:"required"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

// Create handles POST /api/v1/categories/create.
// @Summary      Create a category
// @Tags         categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body categoryCreateRequest true "Category payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/categories/create [post]
func (h *CategoryHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req categoryCreateRequest
	if !bind(c, &req) {
		return
	}
	cat := &model.Category{Name: req.Name, Slug: req.Slug, Description: req.Description,
		ParentID: model.NullableString(req.ParentID), SortOrder: req.SortOrder, IsActive: true}
	cat, err := h.crud.Create(ctx, actor(c), cat)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(cat).Json(c)
}

type categoryUpdateRequest struct {
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	SortOrder   *int   `json:"sort_order,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// Update handles PATCH /api/v1/categories/update.
// @Summary      Update a category
// @Tags         categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body categoryUpdateRequest true "Category update payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/categories/update [patch]
func (h *CategoryHandler) Update(ctx context.Context, c *app.RequestContext) {
	var req categoryUpdateRequest
	if !bind(c, &req) {
		return
	}
	cat := &model.Category{BaseModel: newBaseModel(req.ID), Name: req.Name, Description: req.Description}
	if req.ParentID != "" {
		cat.ParentID = model.NullableString(req.ParentID)
	}
	if req.SortOrder != nil {
		cat.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		cat.IsActive = *req.IsActive
	}
	cat, err := h.crud.Update(ctx, actor(c), cat)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(cat).Json(c)
}

// Delete handles POST /api/v1/categories/delete.
// @Summary      Soft-delete a category
// @Tags         categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Category id"
// @Success      200 {object} response.Response
// @Router       /api/v1/categories/delete [post]
func (h *CategoryHandler) Delete(ctx context.Context, c *app.RequestContext) {
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

// Get handles POST /api/v1/categories/get.
// @Summary      Get a category by id
// @Tags         categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Category id"
// @Success      200 {object} response.Response
// @Router       /api/v1/categories/get [post]
func (h *CategoryHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	cat, err := h.crud.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(cat).Json(c)
}

// List handles POST /api/v1/categories/list.
// @Summary      List categories with cursor pagination
// @Tags         categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/categories/list [post]
func (h *CategoryHandler) List(ctx context.Context, c *app.RequestContext) {
	n, ok := normalize(c, "sort_order")
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
