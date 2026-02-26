package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/meshackkazimoto/elgon"
)

const requestIDKey = "request_id"

// RequestID injects a request identifier in context and response headers.
func RequestID() elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			reqID := c.Header("X-Request-Id")
			if reqID == "" {
				reqID = newRequestID()
			}
			c.Set(requestIDKey, reqID)
			c.Writer.Header().Set("X-Request-Id", reqID)
			return next(c)
		}
	}
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "req_fallback"
	}
	return "req_" + hex.EncodeToString(buf)
}
