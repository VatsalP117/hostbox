package github

import (
	"context"
	"log/slog"
)

// GitHubEventRouter dispatches webhook events to typed handlers.
type GitHubEventRouter struct {
	pushHandler         *PushHandler
	pullRequestHandler  *PullRequestHandler
	installationHandler *InstallationHandler
	logger              *slog.Logger
}

func NewGitHubEventRouter(
	push *PushHandler,
	pr *PullRequestHandler,
	install *InstallationHandler,
	logger *slog.Logger,
) *GitHubEventRouter {
	return &GitHubEventRouter{
		pushHandler:         push,
		pullRequestHandler:  pr,
		installationHandler: install,
		logger:              logger,
	}
}

func (r *GitHubEventRouter) Route(ctx context.Context, eventType string, payload []byte, deliveryID string) error {
	switch eventType {
	case "push":
		return r.pushHandler.Handle(ctx, payload, deliveryID)
	case "pull_request":
		return r.pullRequestHandler.Handle(ctx, payload, deliveryID)
	case "installation":
		return r.installationHandler.Handle(ctx, payload, deliveryID)
	case "installation_repositories":
		return r.installationHandler.HandleRepositories(ctx, payload, deliveryID)
	case "installation_target":
		return r.installationHandler.HandleTarget(ctx, payload, deliveryID)
	case "repository":
		return r.installationHandler.HandleRepository(ctx, payload, deliveryID)
	case "ping":
		r.logger.Info("github ping received", "delivery_id", deliveryID)
		return nil
	default:
		r.logger.Debug("ignoring unhandled github event", "event", eventType)
		return nil
	}
}
