package middleware

import (
	"net/http"

	"github.com/meshackkazimoto/elgon"
)

// BodyLimit rejects payloads larger than maxBytes.
func BodyLimit(maxBytes int64) elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
			return next(c)
		}
	}
}
