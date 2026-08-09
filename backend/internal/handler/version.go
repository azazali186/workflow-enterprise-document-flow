package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// VersionHandler exposes document version endpoints.
type VersionHandler struct {
	crud *service.CrudService[model.Version]
}

// NewVersionHandler wires the handler.
func NewVersionHandler(crud *service.CrudService[model.Version]) *VersionHandler {
	return &VersionHandler{crud: crud}
}

type versionListRequest struct {
	paginationRequest
	DocumentID string `json:"document_id" validate:"required"`
}

// List handles POST /api/v1/versions/list.
// @Summary      List versions of a document
// @Tags         versions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body versionListRequest true "Document id + list payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/versions/list [post]
func (h *VersionHandler) List(ctx context.Context, c *app.RequestContext) {
	var req versionListRequest
	if !bind(c, &req) {
		return
	}
	n, err := req.normalize("version_number")
	if err != nil {
		writeError(c, err)
		return
	}
	n.Filters = map[string]any{"document_id": req.DocumentID}
	items, meta, summary, err := h.crud.List(ctx, n)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(response.Page{Items: items, Pagination: meta, Summary: summary}).Json(c)
}

// Get handles POST /api/v1/versions/get.
// @Summary      Get a version by id
// @Tags         versions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Version id"
// @Success      200 {object} response.Response
// @Router       /api/v1/versions/get [post]
func (h *VersionHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	v, err := h.crud.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(v).Json(c)
}
