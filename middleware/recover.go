package middleware

import (
	"fmt"

	"github.com/meshackkazimoto/elgon"
)

// Recover catches panics and converts them into a typed internal error.
func Recover() elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = elgon.ErrInternal(fmt.Sprint(r))
				}
			}()
			return next(c)
		}
	}
}
