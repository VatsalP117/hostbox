package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/VatsalP117/hostbox/internal/models"
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
	projectRepo      ProjectRepository
	tokenInvalidator InstallationTokenInvalidator
	logger           *slog.Logger
}

func NewInstallationHandler(
	projectRepo ProjectRepository,
	tokenInvalidator InstallationTokenInvalidator,
	logger *slog.Logger,
) *InstallationHandler {
	return &InstallationHandler{
		projectRepo:      projectRepo,
		tokenInvalidator: tokenInvalidator,
		logger:           logger,
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
		if err := h.projectRepo.SetInstallationStatus(ctx, event.Installation.ID, models.GitHubConnectionActive); err != nil {
			return fmt.Errorf("activate installation: %w", err)
		}
	case "deleted":
		h.logger.Info("github app uninstalled",
			"installation_id", event.Installation.ID,
			"account", event.Installation.Account.Login,
		)
		if err := h.projectRepo.ClearInstallation(ctx, event.Installation.ID); err != nil {
			return fmt.Errorf("clear installation: %w", err)
		}
		h.invalidateToken(event.Installation.ID)
	case "suspend", "suspended":
		h.logger.Warn("github app suspended",
			"installation_id", event.Installation.ID,
		)
		if err := h.projectRepo.SetInstallationStatus(ctx, event.Installation.ID, models.GitHubConnectionSuspended); err != nil {
			return fmt.Errorf("suspend installation: %w", err)
		}
		h.invalidateToken(event.Installation.ID)
	case "unsuspend", "unsuspended":
		h.logger.Info("github app unsuspended",
			"installation_id", event.Installation.ID,
		)
		if err := h.projectRepo.SetInstallationStatus(ctx, event.Installation.ID, models.GitHubConnectionActive); err != nil {
			return fmt.Errorf("unsuspend installation: %w", err)
		}
	}

	return nil
}

type InstallationRepositoriesPayload struct {
	Action       string `json:"action"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	RepositoriesAdded   []repositoryIdentity `json:"repositories_added"`
	RepositoriesRemoved []repositoryIdentity `json:"repositories_removed"`
}

type repositoryIdentity struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (h *InstallationHandler) HandleRepositories(ctx context.Context, payload []byte, deliveryID string) error {
	var event InstallationRepositoriesPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal installation_repositories payload: %w", err)
	}

	added := repositoryNames(event.RepositoriesAdded)
	removed := repositoryNames(event.RepositoriesRemoved)
	if err := h.projectRepo.SetRepositoryAccess(ctx, event.Installation.ID, added, true); err != nil {
		return fmt.Errorf("grant repository access: %w", err)
	}
	if err := h.projectRepo.SetRepositoryAccess(ctx, event.Installation.ID, removed, false); err != nil {
		return fmt.Errorf("remove repository access: %w", err)
	}
	if len(removed) > 0 {
		h.invalidateToken(event.Installation.ID)
	}
	h.logger.Info("github installation repository access changed",
		"installation_id", event.Installation.ID,
		"added", len(added),
		"removed", len(removed),
	)
	return nil
}

type RepositoryPayload struct {
	Action       string             `json:"action"`
	Repository   repositoryIdentity `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Changes struct {
		Repository struct {
			Name struct {
				From string `json:"from"`
			} `json:"name"`
		} `json:"repository"`
		Owner struct {
			From struct {
				Login string `json:"login"`
				User  struct {
					Login string `json:"login"`
				} `json:"user"`
			} `json:"from"`
		} `json:"owner"`
	} `json:"changes"`
}

func (h *InstallationHandler) HandleRepository(ctx context.Context, payload []byte, deliveryID string) error {
	var event RepositoryPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal repository payload: %w", err)
	}
	if event.Action != "renamed" && event.Action != "transferred" {
		return nil
	}

	newFullName := normalizedRepositoryName(event.Repository)
	if newFullName == "" || event.Repository.ID <= 0 || event.Installation.ID <= 0 {
		return NewPermanentWebhookError("repository %s payload is missing installation or repository identity", event.Action)
	}

	oldOwner := event.Changes.Owner.From.Login
	if oldOwner == "" {
		oldOwner = event.Changes.Owner.From.User.Login
	}
	oldName := event.Changes.Repository.Name.From
	if oldName == "" {
		oldName = event.Repository.Name
	}
	if oldOwner == "" {
		oldOwner = event.Repository.Owner.Login
	}
	oldFullName := oldOwner + "/" + oldName

	if err := h.projectRepo.UpdateGitHubRepositoryIdentity(
		ctx,
		event.Installation.ID,
		event.Repository.ID,
		oldFullName,
		newFullName,
	); err != nil {
		return fmt.Errorf("follow github repository identity change: %w", err)
	}
	h.logger.Info("github repository identity changed",
		"action", event.Action,
		"repository_id", event.Repository.ID,
		"old_repo", oldFullName,
		"new_repo", newFullName,
	)
	return nil
}

type InstallationTargetPayload struct {
	Action       string `json:"action"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Account struct {
		Login string `json:"login"`
	} `json:"account"`
	Changes struct {
		Login struct {
			From string `json:"from"`
		} `json:"login"`
		Account struct {
			Login struct {
				From string `json:"from"`
			} `json:"login"`
		} `json:"account"`
	} `json:"changes"`
}

func (h *InstallationHandler) HandleTarget(ctx context.Context, payload []byte, deliveryID string) error {
	var event InstallationTargetPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal installation_target payload: %w", err)
	}
	if event.Action != "renamed" {
		return nil
	}
	oldOwner := event.Changes.Login.From
	if oldOwner == "" {
		oldOwner = event.Changes.Account.Login.From
	}
	newOwner := strings.TrimSpace(event.Account.Login)
	if oldOwner == "" || newOwner == "" || event.Installation.ID <= 0 {
		return NewPermanentWebhookError("installation_target rename payload is missing account identity")
	}
	if err := h.projectRepo.RenameInstallationOwner(ctx, event.Installation.ID, oldOwner, newOwner); err != nil {
		return fmt.Errorf("rename github installation owner: %w", err)
	}
	return nil
}

func (h *InstallationHandler) invalidateToken(installationID int64) {
	if h.tokenInvalidator != nil {
		h.tokenInvalidator.InvalidateInstallationToken(installationID)
	}
}

func repositoryNames(repositories []repositoryIdentity) []string {
	names := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		if name := normalizedRepositoryName(repository); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func normalizedRepositoryName(repository repositoryIdentity) string {
	if fullName := strings.TrimSpace(repository.FullName); fullName != "" {
		return fullName
	}
	owner := strings.TrimSpace(repository.Owner.Login)
	name := strings.TrimSpace(repository.Name)
	if owner == "" || name == "" {
		return ""
	}
	return owner + "/" + name
}
