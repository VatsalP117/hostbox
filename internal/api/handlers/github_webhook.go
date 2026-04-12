package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/vatsalpatel/hostbox/internal/services/github"
)

type GitHubWebhookHandler struct {
	webhookSecret []byte
	eventRouter   *github.GitHubEventRouter
	logger        *slog.Logger
}

func NewGitHubWebhookHandler(
	webhookSecret string,
	eventRouter *github.GitHubEventRouter,
	logger *slog.Logger,
) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		webhookSecret: []byte(webhookSecret),
		eventRouter:   eventRouter,
		logger:        logger,
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

	signatureHeader := c.Request().Header.Get("X-Hub-Signature-256")
	if !h.verifySignature(body, signatureHeader) {
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
		if err := h.eventRouter.Route(eventType, body, deliveryID); err != nil {
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

func (h *GitHubWebhookHandler) verifySignature(payload []byte, signatureHeader string) bool {
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	expectedSig := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedBytes, err := hex.DecodeString(expectedSig)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, h.webhookSecret)
	mac.Write(payload)
	computedMAC := mac.Sum(nil)

	return hmac.Equal(computedMAC, expectedBytes)
}
