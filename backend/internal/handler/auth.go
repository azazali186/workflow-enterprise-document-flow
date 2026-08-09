package handler

import (
	"context"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/sessioncookie"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// AuthHandler exposes authentication endpoints.
type AuthHandler struct {
	svc           service.AuthService
	secureCookies bool
}

// NewAuthHandler wires the handler. secureCookies marks session cookies
// Secure (HTTPS-only) — true in production, false in local development.
func NewAuthHandler(svc service.AuthService, secureCookies bool) *AuthHandler {
	return &AuthHandler{svc: svc, secureCookies: secureCookies}
}

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
	h.setSessionCookies(c, result)
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
	h.setSessionCookies(c, result)
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
	// Expire the browser cookies unconditionally, then revoke the SSO entry.
	sessioncookie.Clear(c, h.secureCookies)
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
//
// This route is public (no auth middleware runs), so the token is parsed
// straight from the Authorization header or the session cookie instead of
// the middleware context.
func (h *AuthHandler) Refresh(ctx context.Context, c *app.RequestContext) {
	tokenStr := bearerToken(c)
	if tokenStr == "" {
		tokenStr = sessioncookie.Token(c)
	}
	if tokenStr == "" {
		writeError(c, apperror.Unauthorized("invalid token"))
		return
	}
	claims, err := jwt.ParseToken(tokenStr)
	if err != nil {
		writeError(c, apperror.Unauthorized("invalid token"))
		return
	}
	result, err := h.svc.Refresh(ctx, tokenStr, claims.UserID)
	if err != nil {
		writeError(c, err)
		return
	}
	h.setSessionCookies(c, result)
	response.Success(result).Json(c)
}

// bearerToken extracts a Bearer token from the Authorization header, if any.
func bearerToken(c *app.RequestContext) string {
	authorization := string(c.Request.Header.Peek("Authorization"))
	if len(authorization) > 7 && strings.HasPrefix(authorization, "Bearer ") {
		return authorization[7:]
	}
	return ""
}

// setSessionCookies installs the HttpOnly token cookie. The matching CSRF
// value travels in the response body (AuthResult.CSRF) and lives in the SPA's
// memory; the CSRF middleware recomputes it from this cookie.
func (h *AuthHandler) setSessionCookies(c *app.RequestContext, result *service.AuthResult) {
	sessioncookie.SetToken(c, result.Token, sessioncookie.MaxAgeFor(result.ExpiresAt), h.secureCookies)
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
