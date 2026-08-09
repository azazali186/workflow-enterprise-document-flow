package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// ReportHandler exposes analytics endpoints.
type ReportHandler struct {
	svc service.ReportService
}

// NewReportHandler wires the handler.
func NewReportHandler(svc service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

type dashboardRequest struct {
	Days int `json:"days,omitempty"`
}

// Dashboard handles POST /api/v1/reports/dashboard.
// @Summary      Dashboard aggregates: counts by status, storage, daily trends
// @Tags         reports
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body dashboardRequest true "Days window (default 14)"
// @Success      200 {object} response.Response
// @Router       /api/v1/reports/dashboard [post]
func (h *ReportHandler) Dashboard(ctx context.Context, c *app.RequestContext) {
	var req dashboardRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.svc.Dashboard(ctx, req.Days)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(data).Json(c)
}
