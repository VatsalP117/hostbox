package github

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/VatsalP117/hostbox/internal/models"
)

// --- Mock implementations ---

type mockProjectRepo struct {
	projects map[string]*models.Project // keyed by github_repo
	cleared  []int64
}

func (m *mockProjectRepo) GetByGitHubRepo(ctx context.Context, repo string) (*models.Project, error) {
	p, ok := m.projects[repo]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	return p, nil
}

func (m *mockProjectRepo) ClearInstallation(ctx context.Context, installationID int64) error {
	m.cleared = append(m.cleared, installationID)
	return nil
}

type mockDeploymentCreator struct {
	commits     map[string]*models.Deployment // keyed by "projectID:commitSHA"
	created     []WebhookTriggerParams
	deactivated map[string][]models.Deployment // keyed by "projectID:branch"
}

func (m *mockDeploymentCreator) FindByCommitSHA(ctx context.Context, projectID, commitSHA string) (*models.Deployment, error) {
	key := projectID + ":" + commitSHA
	d, ok := m.commits[key]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	return d, nil
}

func (m *mockDeploymentCreator) CreateFromWebhook(ctx context.Context, params WebhookTriggerParams) (*models.Deployment, error) {
	m.created = append(m.created, params)
	return &models.Deployment{ID: "deploy-1"}, nil
}

func (m *mockDeploymentCreator) DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error) {
	key := projectID + ":" + branch
	if deployments, ok := m.deactivated[key]; ok {
		return deployments, nil
	}
	return nil, nil
}

type mockRouteRemover struct {
	removed []string
}

func (m *mockRouteRemover) RemoveDeploymentRoute(ctx context.Context, deploymentID string) error {
	m.removed = append(m.removed, deploymentID)
	return nil
}

// --- Tests ---

func TestEventRouter_Ping(t *testing.T) {
	router := NewGitHubEventRouter(nil, nil, nil, slog.Default())
	err := router.Route("ping", []byte("{}"), "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventRouter_Unknown(t *testing.T) {
	router := NewGitHubEventRouter(nil, nil, nil, slog.Default())
	err := router.Route("unknown_event", []byte("{}"), "delivery-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushHandler_CreatesDeployment(t *testing.T) {
	proj := &models.Project{
		ID:                 "proj-1",
		ProductionBranch:   "main",
		AutoDeploy:         true,
		PreviewDeployments: true,
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*models.Project{"user/repo": proj},
	}
	deploySvc := &mockDeploymentCreator{
		commits: make(map[string]*models.Deployment),
	}

	handler := NewPushHandler(projectRepo, deploySvc, slog.Default())

	payload, _ := json.Marshal(PushPayload{
		Ref:   "refs/heads/main",
		After: "abc123",
		Repository: struct {
			FullName string `json:"full_name"`
			CloneURL string `json:"clone_url"`
		}{FullName: "user/repo"},
		Installation: struct {
			ID int64 `json:"id"`
		}{ID: 99},
		HeadCommit: struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		}{ID: "abc123", Message: "test commit", Author: struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{Name: "Test User"}},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deploySvc.created) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(deploySvc.created))
	}
	if deploySvc.created[0].Branch != "main" {
		t.Errorf("branch = %q, want main", deploySvc.created[0].Branch)
	}
	if deploySvc.created[0].CommitSHA != "abc123" {
		t.Errorf("commit = %q, want abc123", deploySvc.created[0].CommitSHA)
	}
}

func TestPushHandler_IgnoresBranchDeletion(t *testing.T) {
	handler := NewPushHandler(nil, nil, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"ref":     "refs/heads/main",
		"deleted": true,
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushHandler_IgnoresTagPush(t *testing.T) {
	handler := NewPushHandler(nil, nil, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"ref":   "refs/tags/v1.0",
		"after": "abc123",
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushHandler_IgnoresDisabledAutoDeploy(t *testing.T) {
	proj := &models.Project{
		ID:               "proj-1",
		ProductionBranch: "main",
		AutoDeploy:       false,
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*models.Project{"user/repo": proj},
	}
	deploySvc := &mockDeploymentCreator{}

	handler := NewPushHandler(projectRepo, deploySvc, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"ref":          "refs/heads/main",
		"after":        "abc123",
		"repository":   map[string]string{"full_name": "user/repo"},
		"installation": map[string]int64{"id": 99},
		"head_commit": map[string]interface{}{
			"id":      "abc123",
			"message": "test",
			"author":  map[string]string{"name": "Test"},
		},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deploySvc.created) != 0 {
		t.Error("expected no deployments for disabled auto_deploy")
	}
}

func TestPRHandler_CreatesPreviewDeployment(t *testing.T) {
	proj := &models.Project{
		ID:                 "proj-1",
		ProductionBranch:   "main",
		PreviewDeployments: true,
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*models.Project{"user/repo": proj},
	}
	deploySvc := &mockDeploymentCreator{
		commits: make(map[string]*models.Deployment),
	}
	routes := &mockRouteRemover{}

	handler := NewPullRequestHandler(projectRepo, deploySvc, routes, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "opened",
		"number": 42,
		"pull_request": map[string]interface{}{
			"number": 42,
			"title":  "My PR",
			"state":  "open",
			"head":   map[string]string{"ref": "feature/test", "sha": "def456"},
			"base":   map[string]string{"ref": "main"},
		},
		"repository":   map[string]string{"full_name": "user/repo"},
		"installation": map[string]int64{"id": 99},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deploySvc.created) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(deploySvc.created))
	}
	if deploySvc.created[0].GitHubPRNumber != 42 {
		t.Errorf("PR number = %d, want 42", deploySvc.created[0].GitHubPRNumber)
	}
}

func TestPRHandler_ClosedDeactivatesDeployments(t *testing.T) {
	proj := &models.Project{
		ID:                 "proj-1",
		ProductionBranch:   "main",
		PreviewDeployments: true,
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*models.Project{"user/repo": proj},
	}
	deploySvc := &mockDeploymentCreator{
		deactivated: map[string][]models.Deployment{
			"proj-1:feature/test": {{ID: "deploy-1"}, {ID: "deploy-2"}},
		},
	}
	routes := &mockRouteRemover{}

	handler := NewPullRequestHandler(projectRepo, deploySvc, routes, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "closed",
		"number": 42,
		"pull_request": map[string]interface{}{
			"number": 42,
			"title":  "My PR",
			"state":  "closed",
			"head":   map[string]string{"ref": "feature/test", "sha": "def456"},
			"base":   map[string]string{"ref": "main"},
			"merged": false,
		},
		"repository":   map[string]string{"full_name": "user/repo"},
		"installation": map[string]int64{"id": 99},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes.removed) != 2 {
		t.Errorf("expected 2 route removals, got %d", len(routes.removed))
	}
}

func TestInstallationHandler_Deleted(t *testing.T) {
	projectRepo := &mockProjectRepo{
		projects: make(map[string]*models.Project),
	}

	handler := NewInstallationHandler(projectRepo, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "deleted",
		"installation": map[string]interface{}{
			"id":      99,
			"account": map[string]string{"login": "testuser", "type": "User"},
		},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projectRepo.cleared) != 1 || projectRepo.cleared[0] != 99 {
		t.Error("expected ClearInstallation to be called with 99")
	}
}

func TestCommentManager_BuildCommentBody(t *testing.T) {
	m := &PRCommentManager{
		dashboardDomain: "hostbox.example.com",
	}

	tests := []struct {
		name   string
		info   DeploymentInfo
		expect string
	}{
		{
			name: "ready",
			info: DeploymentInfo{
				ProjectName:   "My App",
				Status:        "ready",
				DeploymentURL: "https://preview.example.com",
				CommitSHA:     "abc1234567890",
				CommitMessage: "fix: something",
				BuildDuration: "45s",
			},
			expect: "Preview Deployment Ready",
		},
		{
			name: "building",
			info: DeploymentInfo{
				ProjectName:   "My App",
				Status:        "building",
				CommitSHA:     "abc1234567890",
				CommitMessage: "fix: something",
			},
			expect: "Preview Deployment Building",
		},
		{
			name: "failed",
			info: DeploymentInfo{
				ProjectName:   "My App",
				Status:        "failed",
				CommitSHA:     "abc1234567890",
				CommitMessage: "fix: something",
				ErrorMessage:  "build timeout",
			},
			expect: "Preview Deployment Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := m.buildCommentBody(tt.info)
			if !contains(body, commentMarker) {
				t.Error("missing comment marker")
			}
			if !contains(body, tt.expect) {
				t.Errorf("body missing expected text %q", tt.expect)
			}
			if !contains(body, "hostbox.example.com") {
				t.Error("missing dashboard domain")
			}
		})
	}
}

func TestStatusReporter_MapStatus(t *testing.T) {
	cases := map[string]string{
		"queued":    "pending",
		"building":  "in_progress",
		"ready":     "success",
		"failed":    "failure",
		"cancelled": "error",
		"unknown":   "error",
	}
	for input, expected := range cases {
		if got := mapStatus(input); got != expected {
			t.Errorf("mapStatus(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestFirstLine(t *testing.T) {
	if got := firstLine("hello\nworld"); got != "hello" {
		t.Errorf("firstLine = %q, want hello", got)
	}
	if got := firstLine("single line"); got != "single line" {
		t.Errorf("firstLine = %q, want single line", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
