package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/vatsalpatel/hostbox/internal/util"
)

const RequestIDHeader = "X-Request-ID"

// RequestID returns middleware that injects a unique request ID.
// Uses X-Request-ID from request header if present, otherwise generates a nanoid.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := c.Request().Header.Get(RequestIDHeader)
			if reqID == "" {
				reqID = util.NewID()
			}
			c.Request().Header.Set(RequestIDHeader, reqID)
			c.Response().Header().Set(RequestIDHeader, reqID)
			c.Set("request_id", reqID)
			return next(c)
		}
	}
}
