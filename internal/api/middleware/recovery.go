package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
)

// Recovery returns middleware that recovers from panics and returns a 500 JSON error.
func Recovery(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					var stack [4096]byte
					n := runtime.Stack(stack[:], false)

					logger.Error("panic recovered",
						"error", fmt.Sprintf("%v", r),
						"stack", string(stack[:n]),
						"path", c.Request().URL.Path,
					)

					if !c.Response().Committed {
						c.JSON(http.StatusInternalServerError, map[string]interface{}{
							"error": map[string]interface{}{
								"code":    "INTERNAL_ERROR",
								"message": "Internal server error",
							},
						})
					}
				}
			}()
			return next(c)
		}
	}
}
