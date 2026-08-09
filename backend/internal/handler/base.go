// Package handler implements the HTTP layer. Every handler follows the same
// contract: JSON body in, unified envelope out, business codes in the body.
// No GET or PUT methods and no path/query parameters are used.
package handler

import (
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// writeError converts any error into the unified envelope.
func writeError(c *app.RequestContext, err error) {
	response.Fail(apperror.MessageOf(err)).SetCode(apperror.CodeOf(err)).Json(c)
}

// actor extracts the authenticated caller from the request context.
func actor(c *app.RequestContext) service.Actor {
	return service.ActorFromHertz(c)
}

// userID returns the authenticated user id (empty for public routes).
func userID(c *app.RequestContext) string {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// normalize validates the pagination part of a list request body.
func normalize(c *app.RequestContext, defaultSort string) (*pagination.Normalized, bool) {
	var req pagination.Request
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, apperror.BadRequest(err.Error()))
		return nil, false
	}
	n, err := req.Normalize(defaultSort)
	if err != nil {
		writeError(c, apperror.BadRequest(err.Error()))
		return nil, false
	}
	return n, true
}

// bind decodes and validates a JSON body into v.
func bind(c *app.RequestContext, v any) bool {
	if err := c.BindAndValidate(v); err != nil {
		writeError(c, apperror.BadRequest(err.Error()))
		return false
	}
	return true
}

// paginationRequest embeds the shared list payload so handlers can extend it.
type paginationRequest struct {
	pagination.Request
}

// normalize delegates to the embedded request normaliser.
func (p *paginationRequest) normalize(defaultSort string) (*pagination.Normalized, error) {
	return p.Normalize(defaultSort)
}

// newBaseModel carries a client-supplied id for partial updates.
func newBaseModel(id string) model.BaseModel {
	return model.BaseModel{ID: id}
}

