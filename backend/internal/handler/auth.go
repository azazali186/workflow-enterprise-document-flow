package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// AuthHandler exposes authentication endpoints.
type AuthHandler struct {
	svc service.AuthService
}

// NewAuthHandler wires the handler.
func NewAuthHandler(svc service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

// registerRequest is the account creation payload.
type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
	Phone    string `json:"phone,omitempty"`
}

// Register handles POST /api/v1/auth/register.
// @Summary      Register a new account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerRequest true "Account details"
// @Success      200 {object} response.Response
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req registerRequest
	if !bind(c, &req) {
		return
	}
	result, err := h.svc.Register(ctx, service.Actor{}, service.RegisterInput{
		Email: req.Email, Password: req.Password, Name: req.Name, Phone: req.Phone,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(result).Json(c)
}

// loginRequest is the authentication payload.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Login handles POST /api/v1/auth/login.
// @Summary      Log in and receive a session token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "Credentials"
// @Success      200 {object} response.Response
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req loginRequest
	if !bind(c, &req) {
		return
	}
	result, err := h.svc.Login(ctx, service.LoginInput{Email: req.Email, Password: req.Password},
		c.ClientIP(), string(c.UserAgent()))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(result).Json(c)
}

// Logout handles POST /api/v1/auth/logout.
// @Summary      Invalidate the active session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(ctx context.Context, c *app.RequestContext) {
	uid := userID(c)
	if uid == "" {
		response.Success(nil).Json(c)
		return
	}
	if err := h.svc.Logout(ctx, actor(c), uid); err != nil {
		writeError(c, err)
		return
	}
	response.Success(nil).Json(c)
}

// Refresh handles POST /api/v1/auth/refresh.
// @Summary      Refresh the session token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(ctx context.Context, c *app.RequestContext) {
	uid := userID(c)
	token, _ := c.Get("token")
	tok, _ := token.(string)
	result, err := h.svc.Refresh(ctx, tok, uid)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(result).Json(c)
}

// Me handles POST /api/v1/auth/me.
// @Summary      Return the current user and roles
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /api/v1/auth/me [post]
func (h *AuthHandler) Me(ctx context.Context, c *app.RequestContext) {
	user, err := h.svc.Me(ctx, userID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(user).Json(c)
}
