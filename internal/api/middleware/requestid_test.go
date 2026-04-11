package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequestID_Generated(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/test", func(c echo.Context) error {
		reqID := c.Get("request_id").(string)
		if reqID == "" {
			t.Error("request_id should be set")
		}
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) == "" {
		t.Error("X-Request-ID response header should be set")
	}
}

func TestRequestID_Preserved(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, "custom-id-123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) != "custom-id-123" {
		t.Errorf("expected preserved request ID, got %q", rec.Header().Get(RequestIDHeader))
	}
}
