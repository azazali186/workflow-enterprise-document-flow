package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// ApprovalHandler exposes approval endpoints.
type ApprovalHandler struct {
	svc service.ApprovalService
}

// NewApprovalHandler wires the handler.
func NewApprovalHandler(svc service.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{svc: svc}
}

type approvalCreateRequest struct {
	DocumentID  string   `json:"document_id" validate:"required"`
	ApproverIDs []string `json:"approver_ids" validate:"required,min=1"`
}

// CreateChain handles POST /api/v1/approvals/create.
// @Summary      Open a multi-level approval chain for a document
// @Tags         approvals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body approvalCreateRequest true "Approval chain payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/approvals/create [post]
func (h *ApprovalHandler) CreateChain(ctx context.Context, c *app.RequestContext) {
	var req approvalCreateRequest
	if !bind(c, &req) {
		return
	}
	chain, err := h.svc.CreateChain(ctx, actor(c), service.CreateApprovalInput{
		DocumentID: req.DocumentID, ApproverIDs: req.ApproverIDs,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(chain).Json(c)
}

type approvalDecideRequest struct {
	ApprovalID string `json:"approval_id" validate:"required"`
	Decision   string `json:"decision" validate:"required,oneof=approved rejected"`
	Comment    string `json:"comment,omitempty"`
}

// Decide handles POST /api/v1/approvals/decide.
// @Summary      Decide an approval step (approved|rejected) with distributed lock
// @Tags         approvals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body approvalDecideRequest true "Decision payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/approvals/decide [post]
func (h *ApprovalHandler) Decide(ctx context.Context, c *app.RequestContext) {
	var req approvalDecideRequest
	if !bind(c, &req) {
		return
	}
	a, err := h.svc.Decide(ctx, actor(c), service.DecideApprovalInput{
		ApprovalID: req.ApprovalID, Decision: req.Decision, Comment: req.Comment,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(a).Json(c)
}

// Get handles POST /api/v1/approvals/get.
// @Summary      Get an approval step by id
// @Tags         approvals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Approval id"
// @Success      200 {object} response.Response
// @Router       /api/v1/approvals/get [post]
func (h *ApprovalHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	a, err := h.svc.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(a).Json(c)
}

// List handles POST /api/v1/approvals/list.
// @Summary      List approvals with cursor pagination
// @Tags         approvals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/approvals/list [post]
func (h *ApprovalHandler) List(ctx context.Context, c *app.RequestContext) {
	n, ok := normalize(c, "created_at")
	if !ok {
		return
	}
	items, meta, summary, err := h.svc.List(ctx, n)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(response.Page{Items: items, Pagination: meta, Summary: summary}).Json(c)
}
