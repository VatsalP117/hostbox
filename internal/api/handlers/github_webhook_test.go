package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

func TestGitHubWebhookHandler_ValidSignature(t *testing.T) {
	secret := "test-secret"
	router := github.NewGitHubEventRouter(nil, nil, nil, slog.Default())
	handler := newWebhookHandlerForTest(t, secret, router)

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
	handler := newWebhookHandlerForTest(t, secret, router)

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
	handler := newWebhookHandlerForTest(t, "secret", router)

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

func newWebhookHandlerForTest(t *testing.T, secret string, router *github.GitHubEventRouter) *GitHubWebhookHandler {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	runtime := github.NewRuntime(slog.Default())
	if err := runtime.Configure(github.AppConfig{
		AppID:         1,
		AppSlug:       "hostbox-test",
		PrivateKeyPEM: keyPEM,
		WebhookSecret: secret,
	}); err != nil {
		t.Fatalf("configure github runtime: %v", err)
	}
	runtime.SetEventRouter(router)

	return NewGitHubWebhookHandler(runtime, slog.Default())
}
