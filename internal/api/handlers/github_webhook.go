package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

type GitHubWebhookHandler struct {
	runtime *github.Runtime
	logger  *slog.Logger
}

func NewGitHubWebhookHandler(
	runtime *github.Runtime,
	logger *slog.Logger,
) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		runtime: runtime,
		logger:  logger,
	}
}

// HandleWebhook processes incoming GitHub webhook events.
func (h *GitHubWebhookHandler) HandleWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{"code": "BAD_REQUEST", "message": "Failed to read request body"},
		})
	}

	webhookSecret, eventRouter, ok := h.runtime.WebhookSecretAndRouter()
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

	h.logger.Info("github webhook received",
		"event", eventType,
		"delivery_id", deliveryID,
	)

	go func() {
		if err := eventRouter.Route(eventType, body, deliveryID); err != nil {
			h.logger.Error("webhook event processing failed",
				"event", eventType,
				"delivery_id", deliveryID,
				"error", err,
			)
		}
	}()

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
