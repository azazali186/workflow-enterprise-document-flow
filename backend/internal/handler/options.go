package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// OptionsHandler serves id+name lookup lists for form dropdowns.
type OptionsHandler struct {
	svc *service.OptionsService
}

// NewOptionsHandler wires the handler.
func NewOptionsHandler(svc *service.OptionsService) *OptionsHandler { return &OptionsHandler{svc: svc} }

// optionsRequest is the lookup payload.
type optionsRequest struct {
	Type   string `json:"type" validate:"required"`
	Search string `json:"search,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// List handles POST /api/v1/options/list.
// @Summary      List id+name options for a dropdown
// @Tags         options
// @Accept       json
// @Produce      json
// @Param        body body optionsRequest true "Option type and optional search"
// @Success      200 {object} response.Response
// @Router       /api/v1/options/list [post]
func (h *OptionsHandler) List(ctx context.Context, c *app.RequestContext) {
	var req optionsRequest
	if !bind(c, &req) {
		return
	}
	options, err := h.svc.List(ctx, req.Type, req.Search, req.Limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(options).Json(c)
}
