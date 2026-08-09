package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// PermissionSyncFn is injected by the bootstrap with the route re-seeder.
var PermissionSyncFn func() (int, error)

// PermissionHandler exposes permission administration endpoints.
type PermissionHandler struct {
	crud *service.CrudService[model.Permission]
}

// NewPermissionHandler wires the handler.
func NewPermissionHandler(crud *service.CrudService[model.Permission]) *PermissionHandler {
	return &PermissionHandler{crud: crud}
}

// Get handles POST /api/v1/permissions/get.
// @Summary      Get a permission by id
// @Tags         permissions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Permission id"
// @Success      200 {object} response.Response
// @Router       /api/v1/permissions/get [post]
func (h *PermissionHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.crud.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(p).Json(c)
}

// List handles POST /api/v1/permissions/list.
// @Summary      List permissions with cursor pagination
// @Tags         permissions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/permissions/list [post]
func (h *PermissionHandler) List(ctx context.Context, c *app.RequestContext) {
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

// Sync handles POST /api/v1/permissions/sync.
// @Summary      Re-scan registered routes and upsert permissions
// @Tags         permissions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /api/v1/permissions/sync [post]
func (h *PermissionHandler) Sync(ctx context.Context, c *app.RequestContext) {
	if PermissionSyncFn == nil {
		writeError(c, apperror.Unavailable("permission sync not wired"))
		return
	}
	n, err := PermissionSyncFn()
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(map[string]any{"permissions_upserted": n}).Json(c)
}
