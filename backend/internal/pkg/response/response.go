// Package response provides the unified API envelope used by every handler.
//
//	response.Fail("bad input").SetCode(400).Json(c)
//	response.Success(data).Json(c)
package response

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Response is the standard envelope. Code is the business code (0 = ok).
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Page carries cursor pagination metadata alongside items.
type Page struct {
	Items         any    `json:"items"`
	Pagination    any    `json:"pagination"`
	Summary       any    `json:"summary,omitempty"`
}

// Success builds an ok response carrying data.
func Success(data any) *Response {
	return &Response{Code: 0, Message: "success", Data: data}
}

// Fail builds an error response. The default business code is 500.
func Fail(message string) *Response {
	return &Response{Code: 500, Message: message}
}

// SetCode overrides the business code (mirrors the classic chained API).
func (r *Response) SetCode(code int) *Response {
	r.Code = code
	return r
}

// SetData attaches payload to an error response.
func (r *Response) SetData(data any) *Response {
	r.Data = data
	return r
}

// Json writes the envelope to the client. The envelope deliberately uses a
// fixed HTTP 200 with the business code in the body; the code is also stashed
// on the request context so the metrics middleware can label the request by
// the real outcome (otherwise HTTP-level alerting could never see errors).
func (r *Response) Json(c *app.RequestContext) {
	c.Set("business_code", r.Code)
	c.JSON(consts.StatusOK, r)
}

