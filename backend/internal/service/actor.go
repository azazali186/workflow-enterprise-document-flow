// Package service implements the application business layer. Services depend
// on repositories and infrastructure through small interfaces so they stay
// unit-testable.
package service

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type ctxKey int

const actorKey ctxKey = 0

// Actor identifies the authenticated caller for audit and ownership.
type Actor struct {
	ID    string
	Email string
	IP    string
	UA    string
}

// WithActor stores the actor in a context.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey, a)
}

// ActorFrom extracts the actor from a context, defaulting to a zero actor.
func ActorFrom(ctx context.Context) Actor {
	if a, ok := ctx.Value(actorKey).(Actor); ok {
		return a
	}
	return Actor{}
}

// ActorFromHertz rebuilds an Actor from a Hertz request context using the
// values written by the auth middleware.
func ActorFromHertz(c *app.RequestContext) Actor {
	a := Actor{}
	if v, ok := c.Get("user_id"); ok {
		a.ID = toString(v)
	}
	if v, ok := c.Get("user_email"); ok {
		a.Email = toString(v)
	}
	a.IP = c.ClientIP()
	a.UA = toString(c.UserAgent())
	return a
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
