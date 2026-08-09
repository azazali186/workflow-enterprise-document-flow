package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// AuditLogHandler exposes audit trail endpoints.
type AuditLogHandler struct {
	crud *service.CrudService[model.AuditLog]
}

// NewAuditLogHandler wires the handler.
func NewAuditLogHandler(crud *service.CrudService[model.AuditLog]) *AuditLogHandler {
	return &AuditLogHandler{crud: crud}
}

// Get handles POST /api/v1/audit-logs/get.
// @Summary      Get an audit entry by id
// @Tags         audit-logs
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Audit id"
// @Success      200 {object} response.Response
// @Router       /api/v1/audit-logs/get [post]
func (h *AuditLogHandler) Get(ctx context.Context, c *app.RequestContext) {
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

// List handles POST /api/v1/audit-logs/list.
// @Summary      List audit entries with filters, date range and summary
// @Tags         audit-logs
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/audit-logs/list [post]
func (h *AuditLogHandler) List(ctx context.Context, c *app.RequestContext) {
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
