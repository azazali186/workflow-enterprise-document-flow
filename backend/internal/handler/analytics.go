package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// AnalyticsHandler exposes the README AnalyticsService module.
type AnalyticsHandler struct {
	svc service.AnalyticsService
}

// NewAnalyticsHandler wires the handler.
func NewAnalyticsHandler(svc service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

type analyticsRequest struct {
	Days int `json:"days,omitempty"` // window, default 14, capped at 90
}

// Documents handles POST /api/v1/analytics/documents.
// @Summary      Documents analytics: by status, by category, per-day trend
// @Tags         analytics
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body analyticsRequest true "Days window (default 14)"
// @Success      200 {object} response.Response
// @Router       /api/v1/analytics/documents [post]
func (h *AnalyticsHandler) Documents(ctx context.Context, c *app.RequestContext) {
	var req analyticsRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.svc.Documents(ctx, req.Days)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(data).Json(c)
}

// Storage handles POST /api/v1/analytics/storage.
// @Summary      Storage analytics: total bytes, by provider, per-day upload trend
// @Tags         analytics
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body analyticsRequest true "Days window (default 14)"
// @Success      200 {object} response.Response
// @Router       /api/v1/analytics/storage [post]
func (h *AnalyticsHandler) Storage(ctx context.Context, c *app.RequestContext) {
	var req analyticsRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.svc.Storage(ctx, req.Days)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(data).Json(c)
}

// Workflow handles POST /api/v1/analytics/workflow.
// @Summary      Workflow analytics: status funnel + pending backlogs
// @Tags         analytics
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body analyticsRequest true "Days window (default 14)"
// @Success      200 {object} response.Response
// @Router       /api/v1/analytics/workflow [post]
func (h *AnalyticsHandler) Workflow(ctx context.Context, c *app.RequestContext) {
	var req analyticsRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.svc.Workflow(ctx, req.Days)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(data).Json(c)
}
