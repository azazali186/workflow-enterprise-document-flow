// Package trace carries a request ID through the HTTP layer, outbox events
// and the worker pipeline so logs and events can be correlated end to end.
package trace

import "context"

type key struct{}

// WithID attaches a request ID to ctx.
func WithID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, key{}, id)
}

// ID returns the request ID carried by ctx, or "" when absent.
func ID(ctx context.Context) string {
	if v, ok := ctx.Value(key{}).(string); ok {
		return v
	}
	return ""
}
