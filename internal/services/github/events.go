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

func (r *GitHubEventRouter) Route(eventType string, payload []byte, deliveryID string) error {
	ctx := context.Background()

	switch eventType {
	case "push":
		return r.pushHandler.Handle(ctx, payload, deliveryID)
	case "pull_request":
		return r.pullRequestHandler.Handle(ctx, payload, deliveryID)
	case "installation":
		return r.installationHandler.Handle(ctx, payload, deliveryID)
	case "ping":
		r.logger.Info("github ping received", "delivery_id", deliveryID)
		return nil
	default:
		r.logger.Debug("ignoring unhandled github event", "event", eventType)
		return nil
	}
}
