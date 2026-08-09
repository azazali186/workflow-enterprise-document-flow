package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// RoleHandler exposes role administration endpoints.
type RoleHandler struct {
	crud *service.CrudService[model.Role]
	rbac service.RBACService
}

// NewRoleHandler wires the handler.
func NewRoleHandler(crud *service.CrudService[model.Role], rbac service.RBACService) *RoleHandler {
	return &RoleHandler{crud: crud, rbac: rbac}
}

type roleCreateRequest struct {
	Code        string `json:"code" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

// Create handles POST /api/v1/roles/create.
// @Summary      Create a role
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body roleCreateRequest true "Role payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/roles/create [post]
func (h *RoleHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req roleCreateRequest
	if !bind(c, &req) {
		return
	}
	role := &model.Role{Code: req.Code, Name: req.Name, Description: req.Description}
	role, err := h.crud.Create(ctx, actor(c), role)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(role).Json(c)
}

type roleUpdateRequest struct {
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// Update handles PATCH /api/v1/roles/update.
// @Summary      Update a role
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body roleUpdateRequest true "Role update payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/roles/update [patch]
func (h *RoleHandler) Update(ctx context.Context, c *app.RequestContext) {
	var req roleUpdateRequest
	if !bind(c, &req) {
		return
	}
	role := &model.Role{BaseModel: newBaseModel(req.ID), Name: req.Name, Description: req.Description}
	role, err := h.crud.Update(ctx, actor(c), role)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(role).Json(c)
}

// Delete handles POST /api/v1/roles/delete.
// @Summary      Soft-delete a role
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Role id"
// @Success      200 {object} response.Response
// @Router       /api/v1/roles/delete [post]
func (h *RoleHandler) Delete(ctx context.Context, c *app.RequestContext) {
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

// Get handles POST /api/v1/roles/get.
// @Summary      Get a role by id
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Role id"
// @Success      200 {object} response.Response
// @Router       /api/v1/roles/get [post]
func (h *RoleHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	role, err := h.crud.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(role).Json(c)
}

// List handles POST /api/v1/roles/list.
// @Summary      List roles with cursor pagination
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/roles/list [post]
func (h *RoleHandler) List(ctx context.Context, c *app.RequestContext) {
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

type assignRequest struct {
	RoleID        string   `json:"role_id" validate:"required"`
	PermissionIDs []string `json:"permission_ids" validate:"required,min=1"`
}

// AssignPermissions handles POST /api/v1/roles/assign-permissions.
// @Summary      Replace a role's permissions
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body assignRequest true "Role + permission ids"
// @Success      200 {object} response.Response
// @Router       /api/v1/roles/assign-permissions [post]
func (h *RoleHandler) AssignPermissions(ctx context.Context, c *app.RequestContext) {
	var req assignRequest
	if !bind(c, &req) {
		return
	}
	if err := h.rbac.AssignPermissions(ctx, req.RoleID, req.PermissionIDs); err != nil {
		writeError(c, err)
		return
	}
	response.Success(nil).Json(c)
}
