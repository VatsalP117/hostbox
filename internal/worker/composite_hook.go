package worker

import (
	"context"

	"github.com/vatsalpatel/hostbox/internal/models"
)

// CompositePostBuildHook chains multiple PostBuildHook implementations.
type CompositePostBuildHook struct {
	hooks []PostBuildHook
}

func NewCompositePostBuildHook(hooks ...PostBuildHook) *CompositePostBuildHook {
	return &CompositePostBuildHook{hooks: hooks}
}

func (c *CompositePostBuildHook) OnBuildSuccess(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	for _, h := range c.hooks {
		if err := h.OnBuildSuccess(ctx, project, deployment); err != nil {
			return err
		}
	}
	return nil
}

func (c *CompositePostBuildHook) OnBuildFailure(ctx context.Context, project *models.Project, deployment *models.Deployment, buildErr error) error {
	for _, h := range c.hooks {
		if err := h.OnBuildFailure(ctx, project, deployment, buildErr); err != nil {
			return err
		}
	}
	return nil
}
