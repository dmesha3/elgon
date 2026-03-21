package observability

import (
	"context"

	"github.com/dmesha3/elgon"
)

// Span is a minimal tracing span abstraction.
type Span interface {
	End()
	RecordError(error)
}

// Tracer starts spans for request execution.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Trace creates middleware that wraps handlers in a span.
func Trace(tracer Tracer) elgon.Middleware {
	if tracer == nil {
		return func(next elgon.HandlerFunc) elgon.HandlerFunc { return next }
	}
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			name := c.Request.Method + " " + c.RoutePattern()
			ctx, span := tracer.Start(c.Request.Context(), name)
			c.Request = c.Request.WithContext(ctx)
			err := next(c)
			if err != nil {
				span.RecordError(err)
			}
			span.End()
			return err
		}
	}
}
