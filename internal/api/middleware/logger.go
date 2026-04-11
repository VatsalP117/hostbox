package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// Logger returns middleware that logs each request with slog.
func Logger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()
			latency := time.Since(start)

			attrs := []slog.Attr{
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.String("ip", c.RealIP()),
			}

			if reqID, ok := c.Get("request_id").(string); ok {
				attrs = append(attrs, slog.String("request_id", reqID))
			}

			level := slog.LevelInfo
			if res.Status >= 500 {
				level = slog.LevelError
			} else if res.Status >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(req.Context(), level, "request",
				attrs...,
			)

			return nil
		}
	}
}
