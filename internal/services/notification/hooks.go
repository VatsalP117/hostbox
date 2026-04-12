package notification

import (
	"context"

	"github.com/vatsalpatel/hostbox/internal/models"
)

// PostBuildNotificationHook implements worker.PostBuildHook to send notifications.
type PostBuildNotificationHook struct {
	service   *Service
	serverURL string
}

func NewPostBuildNotificationHook(service *Service, serverURL string) *PostBuildNotificationHook {
	return &PostBuildNotificationHook{service: service, serverURL: serverURL}
}

func (h *PostBuildNotificationHook) OnBuildSuccess(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	h.service.DispatchForDeployment(ctx, EventDeploySuccess, project, deployment, h.serverURL)
	return nil
}

func (h *PostBuildNotificationHook) OnBuildFailure(ctx context.Context, project *models.Project, deployment *models.Deployment, buildErr error) error {
	h.service.DispatchForDeployment(ctx, EventDeployFailure, project, deployment, h.serverURL)
	return nil
}
