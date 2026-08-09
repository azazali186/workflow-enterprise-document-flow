package handler

import (
	"context"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// SearchHandler exposes the README SearchService module.
type SearchHandler struct {
	svc service.SearchService
}

// NewSearchHandler wires the handler.
func NewSearchHandler(svc service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

type searchRequest struct {
	Search     string `json:"search"` // required keyword
	Status     string `json:"status,omitempty"`
	CategoryID string `json:"category_id,omitempty"`
	OwnerID    string `json:"owner_id,omitempty"`
	pagination.Request
}

// Search handles POST /api/v1/search/documents.
// @Summary      Full-text search over documents with filters + cursor pagination
// @Tags         search
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body searchRequest true "Search payload (search term required)"
// @Success      200 {object} response.Response
// @Router       /api/v1/search/documents [post]
func (h *SearchHandler) Search(ctx context.Context, c *app.RequestContext) {
	var req searchRequest
	if !bind(c, &req) {
		return
	}
	n, err := req.Normalize("created_at")
	if err != nil {
		writeError(c, apperror.BadRequest(err.Error()))
		return
	}
	n.Search = strings.TrimSpace(req.Search)
	if req.Status != "" || req.CategoryID != "" || req.OwnerID != "" {
		filters := map[string]any{}
		if req.Status != "" {
			filters["status"] = req.Status
		}
		if req.CategoryID != "" {
			filters["category_id"] = req.CategoryID
		}
		if req.OwnerID != "" {
			filters["owner_id"] = req.OwnerID
		}
		n.Filters = filters
	}
	items, meta, err := h.svc.SearchDocuments(ctx, actor(c).ID, n)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(response.Page{Items: items, Pagination: meta}).Json(c)
}
