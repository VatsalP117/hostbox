package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

type webhookAcceptorStub struct {
	created    bool
	err        error
	calls      int
	deliveryID string
	eventType  string
	payload    []byte
}

func (s *webhookAcceptorStub) Accept(_ context.Context, deliveryID, eventType string, payload []byte) (bool, error) {
	s.calls++
	s.deliveryID = deliveryID
	s.eventType = eventType
	s.payload = append([]byte(nil), payload...)
	return s.created, s.err
}

func TestGitHubWebhookHandlerValidSignaturePersistsBeforeAccepted(t *testing.T) {
	secret := "test-secret"
	acceptor := &webhookAcceptorStub{created: true}
	handler := newWebhookHandlerForTest(t, secret, acceptor)
	body := `{"action":"ping"}`

	rec := performWebhookRequest(t, handler, body, webhookSignature(secret, body), "ping", "test-delivery")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}
	if acceptor.calls != 1 || acceptor.deliveryID != "test-delivery" || acceptor.eventType != "ping" || string(acceptor.payload) != body {
		t.Fatalf("acceptor call = %#v", acceptor)
	}
}

func TestGitHubWebhookHandlerDuplicateIsAccepted(t *testing.T) {
	secret := "test-secret"
	acceptor := &webhookAcceptorStub{created: false}
	handler := newWebhookHandlerForTest(t, secret, acceptor)
	body := `{}`

	rec := performWebhookRequest(t, handler, body, webhookSignature(secret, body), "ping", "duplicate")
	if rec.Code != http.StatusAccepted || acceptor.calls != 1 {
		t.Fatalf("duplicate status=%d calls=%d", rec.Code, acceptor.calls)
	}
}

func TestGitHubWebhookHandlerInvalidSignatureIsNotStored(t *testing.T) {
	acceptor := &webhookAcceptorStub{created: true}
	handler := newWebhookHandlerForTest(t, "test-secret", acceptor)

	rec := performWebhookRequest(t, handler, `{}`, "sha256=invalid", "ping", "delivery")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if acceptor.calls != 0 {
		t.Fatalf("acceptor calls = %d, want 0", acceptor.calls)
	}
}

func TestGitHubWebhookHandlerMissingSignatureIsNotStored(t *testing.T) {
	acceptor := &webhookAcceptorStub{created: true}
	handler := newWebhookHandlerForTest(t, "secret", acceptor)

	rec := performWebhookRequest(t, handler, `{}`, "", "ping", "delivery")
	if rec.Code != http.StatusUnauthorized || acceptor.calls != 0 {
		t.Fatalf("status=%d calls=%d, want 401/0", rec.Code, acceptor.calls)
	}
}

func TestGitHubWebhookHandlerRequiresDeliveryHeaders(t *testing.T) {
	secret := "secret"
	acceptor := &webhookAcceptorStub{created: true}
	handler := newWebhookHandlerForTest(t, secret, acceptor)
	body := `{}`

	rec := performWebhookRequest(t, handler, body, webhookSignature(secret, body), "ping", "")
	if rec.Code != http.StatusBadRequest || acceptor.calls != 0 {
		t.Fatalf("status=%d calls=%d, want 400/0", rec.Code, acceptor.calls)
	}
}

func TestGitHubWebhookHandlerRejectsOversizedPayload(t *testing.T) {
	acceptor := &webhookAcceptorStub{created: true}
	handler := newWebhookHandlerForTest(t, "secret", acceptor)
	body := strings.Repeat("x", int(github.MaxWebhookPayloadBytes)+1)

	rec := performWebhookRequest(t, handler, body, "sha256=unused", "push", "delivery")
	if rec.Code != http.StatusRequestEntityTooLarge || acceptor.calls != 0 {
		t.Fatalf("status=%d calls=%d, want 413/0", rec.Code, acceptor.calls)
	}
}

func TestGitHubWebhookHandlerPersistenceFailureIsNotAcknowledged(t *testing.T) {
	secret := "secret"
	acceptor := &webhookAcceptorStub{err: errors.New("database unavailable")}
	handler := newWebhookHandlerForTest(t, secret, acceptor)
	body := `{}`

	rec := performWebhookRequest(t, handler, body, webhookSignature(secret, body), "ping", "delivery")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
}

func newWebhookHandlerForTest(t *testing.T, secret string, acceptor GitHubWebhookDeliveryAcceptor) *GitHubWebhookHandler {
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
	runtime.SetEventRouter(github.NewGitHubEventRouter(nil, nil, nil, slog.Default()))

	return NewGitHubWebhookHandler(runtime, acceptor, slog.Default())
}

func performWebhookRequest(
	t *testing.T,
	handler *GitHubWebhookHandler,
	body, signature, eventType, deliveryID string,
) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-Hub-Signature-256", signature)
	}
	if eventType != "" {
		req.Header.Set("X-GitHub-Event", eventType)
	}
	if deliveryID != "" {
		req.Header.Set("X-GitHub-Delivery", deliveryID)
	}
	rec := httptest.NewRecorder()
	if err := handler.HandleWebhook(e.NewContext(req, rec)); err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	return rec
}

func webhookSignature(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
