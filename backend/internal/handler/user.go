package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// UserHandler exposes user administration endpoints.
type UserHandler struct {
	svc service.UserService
}

// NewUserHandler wires the handler.
func NewUserHandler(svc service.UserService) *UserHandler { return &UserHandler{svc: svc} }

type userCreateRequest struct {
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required,min=8"`
	Name     string   `json:"name" validate:"required"`
	Phone    string   `json:"phone,omitempty"`
	RoleIDs  []string `json:"role_ids,omitempty"`
}

// Create handles POST /api/v1/users/create.
// @Summary      Create a user
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body userCreateRequest true "User payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/users/create [post]
func (h *UserHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req userCreateRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.Create(ctx, actor(c), service.CreateUserInput{
		Email: req.Email, Password: req.Password, Name: req.Name, Phone: req.Phone, RoleIDs: req.RoleIDs,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(user).Json(c)
}

type userUpdateRequest struct {
	ID      string   `json:"id" validate:"required"`
	Name    string   `json:"name,omitempty"`
	Phone   string   `json:"phone,omitempty"`
	Status  string   `json:"status,omitempty"`
	RoleIDs []string `json:"role_ids,omitempty"`
}

// Update handles PATCH /api/v1/users/update.
// @Summary      Update a user
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body userUpdateRequest true "User update payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/users/update [patch]
func (h *UserHandler) Update(ctx context.Context, c *app.RequestContext) {
	var req userUpdateRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.Update(ctx, actor(c), service.UpdateUserInput{
		ID: req.ID, Name: req.Name, Phone: req.Phone, Status: req.Status, RoleIDs: req.RoleIDs,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(user).Json(c)
}

type idRequest struct {
	ID string `json:"id" validate:"required"`
}

// Delete handles POST /api/v1/users/delete.
// @Summary      Soft-delete a user
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "User id"
// @Success      200 {object} response.Response
// @Router       /api/v1/users/delete [post]
func (h *UserHandler) Delete(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.Delete(ctx, actor(c), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(nil).Json(c)
}

// Get handles POST /api/v1/users/get.
// @Summary      Get a user by id
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body idRequest true "User id"
// @Success      200 {object} response.Response
// @Router       /api/v1/users/get [post]
func (h *UserHandler) Get(ctx context.Context, c *app.RequestContext) {
	var req idRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.Get(ctx, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(user).Json(c)
}

// List handles POST /api/v1/users/list.
// @Summary      List users with cursor pagination
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body pagination.Request true "List payload"
// @Success      200 {object} response.Response
// @Router       /api/v1/users/list [post]
func (h *UserHandler) List(ctx context.Context, c *app.RequestContext) {
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
