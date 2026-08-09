package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// LoginLogHandler exposes authentication log endpoints.
type LoginLogHandler struct {
	crud *service.CrudService[model.LoginLog]
}

// NewLoginLogHandler wires the handler.
func NewLoginLogHandler(crud *service.CrudService[model.LoginLog]) *LoginLogHandler {
	return &LoginLogHandler{crud: crud}
}

// Get handles POST /api/v1/login-logs/get.
// @Summary      Get a login log entry by id
// @Tags         login-logs
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Log id"
// @Success      200 {object} response.Response
// @Router       /api/v1/login-logs/get [post]
func (h *LoginLogHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	entry, err := h.crud.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(entry).Json(c)
}

// List handles POST /api/v1/login-logs/list.
// @Summary      List login attempts with filters, date range and summary
// @Tags         login-logs
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/login-logs/list [post]
func (h *LoginLogHandler) List(ctx context.Context, c *app.RequestContext) {
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
