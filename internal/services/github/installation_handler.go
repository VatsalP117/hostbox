package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// InstallationPayload is the GitHub installation webhook payload.
type InstallationPayload struct {
	Action       string `json:"action"`
	Installation struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"account"`
	} `json:"installation"`
	Repositories []struct {
		FullName string `json:"full_name"`
	} `json:"repositories"`
}

type InstallationHandler struct {
	projectRepo ProjectRepository
	logger      *slog.Logger
}

func NewInstallationHandler(
	projectRepo ProjectRepository,
	logger *slog.Logger,
) *InstallationHandler {
	return &InstallationHandler{
		projectRepo: projectRepo,
		logger:      logger,
	}
}

func (h *InstallationHandler) Handle(ctx context.Context, payload []byte, deliveryID string) error {
	var event InstallationPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal installation payload: %w", err)
	}

	switch event.Action {
	case "created":
		h.logger.Info("github app installed",
			"installation_id", event.Installation.ID,
			"account", event.Installation.Account.Login,
		)
	case "deleted":
		h.logger.Info("github app uninstalled",
			"installation_id", event.Installation.ID,
			"account", event.Installation.Account.Login,
		)
		if err := h.projectRepo.ClearInstallation(ctx, event.Installation.ID); err != nil {
			return fmt.Errorf("clear installation: %w", err)
		}
	case "suspend":
		h.logger.Warn("github app suspended",
			"installation_id", event.Installation.ID,
		)
	case "unsuspend":
		h.logger.Info("github app unsuspended",
			"installation_id", event.Installation.ID,
		)
	}

	return nil
}
