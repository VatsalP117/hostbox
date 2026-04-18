package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/VatsalP117/hostbox/internal/services/github"
)

func TestGitHubWebhookHandler_ValidSignature(t *testing.T) {
	secret := "test-secret"
	router := github.NewGitHubEventRouter(nil, nil, nil, slog.Default())
	handler := NewGitHubWebhookHandler(secret, router, slog.Default())

	body := `{"action":"ping"}`
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-GitHub-Delivery", "test-delivery")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleWebhook(c)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", rec.Code)
	}
}

func TestGitHubWebhookHandler_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	router := github.NewGitHubEventRouter(nil, nil, nil, slog.Default())
	handler := NewGitHubWebhookHandler(secret, router, slog.Default())

	body := `{"action":"ping"}`

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	req.Header.Set("X-GitHub-Event", "ping")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleWebhook(c)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestGitHubWebhookHandler_MissingSignature(t *testing.T) {
	router := github.NewGitHubEventRouter(nil, nil, nil, slog.Default())
	handler := NewGitHubWebhookHandler("secret", router, slog.Default())

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleWebhook(c)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
