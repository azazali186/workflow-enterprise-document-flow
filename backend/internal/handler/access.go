package handler

import (
	"context"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// AccessHandler exposes document access endpoints.
type AccessHandler struct {
	crud *service.CrudService[model.Access]
}

// NewAccessHandler wires the handler.
func NewAccessHandler(crud *service.CrudService[model.Access]) *AccessHandler {
	return &AccessHandler{crud: crud}
}

type accessGrantRequest struct {
	DocumentID string `json:"document_id" validate:"required"`
	UserID     string `json:"user_id,omitempty"`
	RoleID     string `json:"role_id,omitempty"`
	Permission string `json:"permission" validate:"required,oneof=read write approve"`
}

// Grant handles POST /api/v1/accesses/grant.
// @Summary      Grant document access to a user or role
// @Tags         accesses
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body accessGrantRequest true "Access grant payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/accesses/grant [post]
func (h *AccessHandler) Grant(ctx context.Context, c *app.RequestContext) {
	var req accessGrantRequest
	if !bind(c, &req) {
		return
	}
	if req.UserID == "" && req.RoleID == "" {
		writeError(c, apperror.BadRequest("either user_id or role_id is required"))
		return
	}
	a := &model.Access{
		DocumentID: req.DocumentID, UserID: model.NullableString(req.UserID),
		RoleID: model.NullableString(req.RoleID), Permission: req.Permission, GrantedBy: userID(c),
	}
	a, err := h.crud.Create(ctx, actor(c), a)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(a).Json(c)
}

type accessRevokeRequest struct {
	AccessID string `json:"access_id" validate:"required"`
}

// Revoke handles POST /api/v1/accesses/revoke.
// @Summary      Revoke document access
// @Tags         accesses
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body accessRevokeRequest true "Access id"
// @Success      200 {object} response.Response
// @Router       /api/v1/accesses/revoke [post]
func (h *AccessHandler) Revoke(ctx context.Context, c *app.RequestContext) {
	var req accessRevokeRequest
	if !bind(c, &req) {
		return
	}
	existing, err := h.crud.Get(ctx, req.AccessID)
	if err != nil {
		writeError(c, err)
		return
	}
	now := time.Now()
	existing.RevokedAt = &now
	existing, err = h.crud.Update(ctx, actor(c), existing)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(existing).Json(c)
}

// List handles POST /api/v1/accesses/list.
// @Summary      List access grants with cursor pagination
// @Tags         accesses
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/accesses/list [post]
func (h *AccessHandler) List(ctx context.Context, c *app.RequestContext) {
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


