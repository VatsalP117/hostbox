package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// CORS returns Echo middleware restricting cross-origin requests to the platform domain.
func CORS(platformDomain string, useHTTPS bool) echo.MiddlewareFunc {
	scheme := "https"
	if !useHTTPS {
		scheme = "http"
	}
	origin := fmt.Sprintf("%s://%s", scheme, platformDomain)

	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     []string{origin},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           86400,
	})
}
