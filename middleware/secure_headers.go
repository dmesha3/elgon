package middleware

import "github.com/dmesha3/elgon"

// SecureHeaders sets baseline secure HTTP response headers.
func SecureHeaders() elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			h := c.Writer.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			return next(c)
		}
	}
}
