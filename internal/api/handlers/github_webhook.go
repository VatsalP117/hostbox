package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

type GitHubWebhookHandler struct {
	runtime   *github.Runtime
	processor GitHubWebhookDeliveryAcceptor
	logger    *slog.Logger
}

type GitHubWebhookDeliveryAcceptor interface {
	Accept(ctx context.Context, deliveryID, eventType string, payload []byte) (created bool, err error)
}

func NewGitHubWebhookHandler(
	runtime *github.Runtime,
	processor GitHubWebhookDeliveryAcceptor,
	logger *slog.Logger,
) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		runtime:   runtime,
		processor: processor,
		logger:    logger,
	}
}

// HandleWebhook processes incoming GitHub webhook events.
func (h *GitHubWebhookHandler) HandleWebhook(c echo.Context) error {
	body, err := io.ReadAll(http.MaxBytesReader(c.Response(), c.Request().Body, github.MaxWebhookPayloadBytes))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return c.JSON(http.StatusRequestEntityTooLarge, map[string]any{
				"error": map[string]string{"code": "PAYLOAD_TOO_LARGE", "message": "Webhook payload is too large"},
			})
		}
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{"code": "BAD_REQUEST", "message": "Failed to read request body"},
		})
	}

	webhookSecret, _, ok := h.runtime.WebhookSecretAndRouter()
	if !ok {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{"code": "GITHUB_NOT_READY", "message": "GitHub webhook handling is not ready"},
		})
	}

	signatureHeader := c.Request().Header.Get("X-Hub-Signature-256")
	if !h.verifySignature(body, signatureHeader, webhookSecret) {
		h.logger.Warn("webhook signature verification failed",
			"delivery_id", c.Request().Header.Get("X-GitHub-Delivery"),
		)
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"code": "UNAUTHORIZED", "message": "Invalid webhook signature"},
		})
	}

	eventType := c.Request().Header.Get("X-GitHub-Event")
	deliveryID := c.Request().Header.Get("X-GitHub-Delivery")
	if eventType == "" || deliveryID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{"code": "BAD_REQUEST", "message": "GitHub event and delivery headers are required"},
		})
	}
	if len(eventType) > 100 || len(deliveryID) > 255 {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{"code": "BAD_REQUEST", "message": "GitHub event or delivery header is too long"},
		})
	}
	if h.processor == nil {
		h.logger.Error("github webhook processor is not configured")
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{"code": "GITHUB_NOT_READY", "message": "GitHub webhook handling is not ready"},
		})
	}

	created, err := h.processor.Accept(c.Request().Context(), deliveryID, eventType, body)
	if err != nil {
		h.logger.Error("failed to persist github webhook delivery",
			"event", eventType,
			"delivery_id", deliveryID,
			"error", err,
		)
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{"code": "WEBHOOK_PERSIST_FAILED", "message": "Failed to accept webhook delivery"},
		})
	}

	h.logger.Info("github webhook accepted",
		"event", eventType,
		"delivery_id", deliveryID,
		"duplicate", !created,
	)

	return c.JSON(http.StatusAccepted, map[string]any{
		"received": true,
	})
}

func (h *GitHubWebhookHandler) verifySignature(payload []byte, signatureHeader string, webhookSecret string) bool {
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	expectedSig := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedBytes, err := hex.DecodeString(expectedSig)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(payload)
	computedMAC := mac.Sum(nil)

	return hmac.Equal(computedMAC, expectedBytes)
}
