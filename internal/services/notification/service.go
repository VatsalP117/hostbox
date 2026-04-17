package notification

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
)

// Event types
const (
	EventDeploySuccess    = "deploy_success"
	EventDeployFailure    = "deploy_failure"
	EventDomainVerified   = "domain_verified"
	EventDomainUnverified = "domain_unverified"
)

type NotificationClient interface {
	Send(ctx context.Context, webhookURL string, payload NotificationPayload) error
}

type NotificationPayload struct {
	Event      string          `json:"event"`
	Project    ProjectInfo     `json:"project"`
	Deployment *DeploymentInfo `json:"deployment,omitempty"`
	Domain     *DomainInfo     `json:"domain,omitempty"`
	Timestamp  string          `json:"timestamp"`
	ServerURL  string          `json:"server_url"`
}

type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type DeploymentInfo struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	Branch          string `json:"branch"`
	CommitSHA       string `json:"commit_sha"`
	CommitMessage   string `json:"commit_message,omitempty"`
	CommitAuthor    string `json:"commit_author,omitempty"`
	DeploymentURL   string `json:"deployment_url,omitempty"`
	DashboardURL    string `json:"dashboard_url,omitempty"`
	BuildDurationMs int64  `json:"build_duration_ms,omitempty"`
	IsProduction    bool   `json:"is_production"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

type DomainInfo struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	Verified bool   `json:"verified"`
}

type Service struct {
	repo    *repository.NotificationRepository
	clients map[string]NotificationClient
	logger  *slog.Logger
}

func NewService(repo *repository.NotificationRepository, logger *slog.Logger) *Service {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return &Service{
		repo: repo,
		clients: map[string]NotificationClient{
			"discord": &DiscordClient{httpClient: httpClient},
			"slack":   &SlackClient{httpClient: httpClient},
			"webhook": &WebhookClient{httpClient: httpClient},
		},
		logger: logger,
	}
}

func (s *Service) Dispatch(ctx context.Context, event string, payload NotificationPayload) {
	payload.Event = event
	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)

	configs, err := s.repo.FindByProjectAndEvent(ctx, payload.Project.ID, event)
	if err != nil {
		s.logger.Error("failed to fetch project notification configs", "error", err)
	}

	globalConfigs, err := s.repo.FindGlobalByEvent(ctx, event)
	if err != nil {
		s.logger.Error("failed to fetch global notification configs", "error", err)
	}
	configs = append(configs, globalConfigs...)

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		client, ok := s.clients[cfg.Channel]
		if !ok {
			s.logger.Error("unknown notification channel", "channel", cfg.Channel)
			continue
		}

		go s.sendWithRetry(context.Background(), client, cfg.WebhookURL, payload, 3)
	}
}

// DispatchForDeployment is a convenience method that builds a payload from model objects.
func (s *Service) DispatchForDeployment(ctx context.Context, event string, project *models.Project, deployment *models.Deployment, serverURL string) {
	payload := NotificationPayload{
		Project: ProjectInfo{
			ID:   project.ID,
			Name: project.Name,
			Slug: project.Slug,
		},
		ServerURL: serverURL,
	}

	if deployment != nil {
		di := &DeploymentInfo{
			ID:           deployment.ID,
			Status:       string(deployment.Status),
			Branch:       deployment.Branch,
			CommitSHA:    deployment.CommitSHA,
			IsProduction: deployment.IsProduction,
		}
		if deployment.CommitMessage != nil {
			di.CommitMessage = *deployment.CommitMessage
		}
		if deployment.CommitAuthor != nil {
			di.CommitAuthor = *deployment.CommitAuthor
		}
		if deployment.DeploymentURL != nil {
			di.DeploymentURL = *deployment.DeploymentURL
		}
		if deployment.ErrorMessage != nil {
			di.ErrorMessage = *deployment.ErrorMessage
		}
		if deployment.BuildDurationMs != nil {
			di.BuildDurationMs = *deployment.BuildDurationMs
		}
		payload.Deployment = di
	}

	s.Dispatch(ctx, event, payload)
}

func (s *Service) sendWithRetry(
	ctx context.Context,
	client NotificationClient,
	webhookURL string,
	payload NotificationPayload,
	maxRetries int,
) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := client.Send(ctx, webhookURL, payload)
		if err == nil {
			return
		}

		s.logger.Warn("notification send failed",
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"event", payload.Event,
			"error", err,
		)

		if attempt < maxRetries {
			backoff := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}

	s.logger.Error("notification permanently failed after retries",
		"event", payload.Event,
		"project", payload.Project.Slug,
	)
}

func (s *Service) SendTest(ctx context.Context, config *models.NotificationConfig, payload NotificationPayload) error {
	client, ok := s.clients[config.Channel]
	if !ok {
		return fmt.Errorf("unknown notification channel %q", config.Channel)
	}

	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return client.Send(ctx, config.WebhookURL, payload)
}
