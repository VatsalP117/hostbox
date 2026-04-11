package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/vatsalpatel/hostbox/internal/models"
)

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := &RateLimiter{config: RateLimiterConfig{Rate: 10, Burst: 10}}

	for i := 0; i < 10; i++ {
		allowed, _, _ := rl.Allow("test-key")
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksExcess(t *testing.T) {
	rl := &RateLimiter{config: RateLimiterConfig{Rate: 5, Burst: 5}}

	// Exhaust burst
	for i := 0; i < 5; i++ {
		rl.Allow("test-key")
	}

	allowed, _, _ := rl.Allow("test-key")
	if allowed {
		t.Error("6th request should be blocked")
	}
}

func TestRateLimiter_SeparateKeys(t *testing.T) {
	rl := &RateLimiter{config: RateLimiterConfig{Rate: 2, Burst: 2}}

	// Exhaust key A
	rl.Allow("A")
	rl.Allow("A")
	allowed, _, _ := rl.Allow("A")
	if allowed {
		t.Error("key A should be blocked")
	}

	// Key B should still work
	allowed, _, _ = rl.Allow("B")
	if !allowed {
		t.Error("key B should be allowed")
	}
}

func TestRateLimitMiddleware_SetsHeaders(t *testing.T) {
	rl := &RateLimiter{config: RateLimiterConfig{Rate: 10, Burst: 10}}
	e := echo.New()

	handler := RateLimit(rl, IPKeyFunc)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("X-RateLimit-Limit = %q, want 10", rec.Header().Get("X-RateLimit-Limit"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("X-RateLimit-Remaining should be set")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset should be set")
	}
}

func TestRateLimitMiddleware_Returns429(t *testing.T) {
	rl := &RateLimiter{config: RateLimiterConfig{Rate: 1, Burst: 1}}
	e := echo.New()

	handler := RateLimit(rl, IPKeyFunc)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	// First request passes
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}

	// Second request gets rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	err := handler(c2)
	if err == nil {
		t.Fatal("second request should be rate limited")
	}
}

func TestSecurityHeaders(t *testing.T) {
	e := echo.New()
	handler := SecurityHeaders()(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=()",
	}

	for header, want := range expected {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestCORS(t *testing.T) {
	e := echo.New()
	handler := CORS("hostbox.example.com", true)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://hostbox.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://hostbox.example.com" {
		t.Errorf("origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("credentials should be allowed")
	}
}

func TestIPKeyFunc(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	key := IPKeyFunc(c)
	if key == "" {
		t.Error("IP key should not be empty")
	}
}

func TestUserKeyFunc_WithUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(UserContextKey), &models.User{ID: "user-123"})

	key := UserKeyFunc(c)
	if key != "user-123" {
		t.Errorf("key = %q, want user-123", key)
	}
}

func TestUserKeyFunc_WithoutUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	key := UserKeyFunc(c)
	// Should fall back to IP
	if key == "" {
		t.Error("should fall back to IP")
	}
}
