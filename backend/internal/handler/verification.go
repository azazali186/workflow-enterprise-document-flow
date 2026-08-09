package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// VerificationHandler exposes verification endpoints.
type VerificationHandler struct {
	svc service.VerificationService
}

// NewVerificationHandler wires the handler.
func NewVerificationHandler(svc service.VerificationService) *VerificationHandler {
	return &VerificationHandler{svc: svc}
}

type verificationCreateRequest struct {
	DocumentID string `json:"document_id" validate:"required"`
	Method     string `json:"method,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// Create handles POST /api/v1/verifications/create.
// @Summary      Start a document verification request
// @Tags         verifications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body verificationCreateRequest true "Verification payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/verifications/create [post]
func (h *VerificationHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req verificationCreateRequest
	if !bind(c, &req) {
		return
	}
	ver, err := h.svc.Create(ctx, actor(c), service.CreateVerificationInput{
		DocumentID: req.DocumentID, Method: req.Method, Notes: req.Notes,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(ver).Json(c)
}

type verificationDecideRequest struct {
	VerificationID string `json:"verification_id" validate:"required"`
	Decision       string `json:"decision" validate:"required,oneof=verified rejected"`
	Notes          string `json:"notes,omitempty"`
}

// Decide handles POST /api/v1/verifications/decide.
// @Summary      Decide a verification (verified|rejected) with distributed lock
// @Tags         verifications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body verificationDecideRequest true "Decision payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/verifications/decide [post]
func (h *VerificationHandler) Decide(ctx context.Context, c *app.RequestContext) {
	var req verificationDecideRequest
	if !bind(c, &req) {
		return
	}
	ver, err := h.svc.Decide(ctx, actor(c), service.DecideVerificationInput{
		VerificationID: req.VerificationID, Decision: req.Decision, Notes: req.Notes,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(ver).Json(c)
}

// Get handles POST /api/v1/verifications/get.
// @Summary      Get a verification by id
// @Tags         verifications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "Verification id"
// @Success      200 {object} response.Response
// @Router       /api/v1/verifications/get [post]
func (h *VerificationHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	ver, err := h.svc.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(ver).Json(c)
}

// List handles POST /api/v1/verifications/list.
// @Summary      List verifications with cursor pagination
// @Tags         verifications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/verifications/list [post]
func (h *VerificationHandler) List(ctx context.Context, c *app.RequestContext) {
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
