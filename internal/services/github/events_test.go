package github

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/VatsalP117/hostbox/internal/models"
)

// --- Mock implementations ---

type mockProjectRepo struct {
	projects          map[string]*models.Project // keyed by github_repo
	projectLists      map[string][]models.Project
	cleared           []int64
	statuses          map[int64]string
	accessChanges     []repositoryAccessChange
	identityChanges   []repositoryIdentityChange
	installationMoves []installationOwnerChange
}

type repositoryAccessChange struct {
	installationID int64
	repositories   []string
	granted        bool
}

type repositoryIdentityChange struct {
	installationID int64
	repositoryID   int64
	oldFullName    string
	newFullName    string
}

type installationOwnerChange struct {
	installationID int64
	oldOwner       string
	newOwner       string
}

func (m *mockProjectRepo) ListByGitHubSource(ctx context.Context, installationID int64, repo string) ([]models.Project, error) {
	if projects, ok := m.projectLists[repo]; ok {
		return projects, nil
	}
	p, ok := m.projects[repo]
	if !ok {
		return nil, nil
	}
	return []models.Project{*p}, nil
}

func (m *mockProjectRepo) ClearInstallation(ctx context.Context, installationID int64) error {
	m.cleared = append(m.cleared, installationID)
	return nil
}

func (m *mockProjectRepo) SetInstallationStatus(ctx context.Context, installationID int64, status string) error {
	if m.statuses == nil {
		m.statuses = make(map[int64]string)
	}
	m.statuses[installationID] = status
	return nil
}

func (m *mockProjectRepo) SetRepositoryAccess(ctx context.Context, installationID int64, repos []string, granted bool) error {
	m.accessChanges = append(m.accessChanges, repositoryAccessChange{installationID, append([]string(nil), repos...), granted})
	return nil
}

func (m *mockProjectRepo) UpdateGitHubRepositoryIdentity(ctx context.Context, installationID, repositoryID int64, oldFullName, newFullName string) error {
	m.identityChanges = append(m.identityChanges, repositoryIdentityChange{installationID, repositoryID, oldFullName, newFullName})
	return nil
}

func (m *mockProjectRepo) RenameInstallationOwner(ctx context.Context, installationID int64, oldOwner, newOwner string) error {
	m.installationMoves = append(m.installationMoves, installationOwnerChange{installationID, oldOwner, newOwner})
	return nil
}

type mockDeploymentCreator struct {
	commits       map[string]*models.Deployment // keyed by "projectID:commitSHA"
	created       []WebhookTriggerParams
	deactivated   map[string][]models.Deployment // keyed by "projectID:branch"
	deactivateErr error
	associated    []int
}

func (m *mockDeploymentCreator) FindByCommitSHAAndBranch(ctx context.Context, projectID, commitSHA, branch string) (*models.Deployment, error) {
	key := projectID + ":" + branch + ":" + commitSHA
	d, ok := m.commits[key]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	return d, nil
}

func (m *mockDeploymentCreator) AssociatePullRequest(_ context.Context, deployment *models.Deployment, prNumber int) error {
	m.associated = append(m.associated, prNumber)
	deployment.GitHubPRNumber = &prNumber
	return nil
}

func (m *mockDeploymentCreator) CreateFromWebhook(ctx context.Context, params WebhookTriggerParams) (*models.Deployment, error) {
	m.created = append(m.created, params)
	deployment := &models.Deployment{
		ID: "deploy-1", ProjectID: params.ProjectID, Branch: params.Branch,
		CommitSHA: params.CommitSHA, Status: models.DeploymentStatusQueued,
	}
	if params.GitHubPRNumber > 0 {
		prNumber := params.GitHubPRNumber
		deployment.GitHubPRNumber = &prNumber
	}
	if m.commits == nil {
		m.commits = make(map[string]*models.Deployment)
	}
	m.commits[params.ProjectID+":"+params.Branch+":"+params.CommitSHA] = deployment
	return deployment, nil
}

func (m *mockDeploymentCreator) DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error) {
	if m.deactivateErr != nil {
		return nil, m.deactivateErr
	}
	key := projectID + ":" + branch
	if deployments, ok := m.deactivated[key]; ok {
		return deployments, nil
	}
	return nil, nil
}

type mockRouteRemover struct {
	removed          []string
	removedBranches  []string
	deploymentErrors map[string]error
	branchError      error
}

type mockTokenInvalidator struct {
	installationIDs []int64
}

func (m *mockTokenInvalidator) InvalidateInstallationToken(installationID int64) {
	m.installationIDs = append(m.installationIDs, installationID)
}

func (m *mockRouteRemover) RemoveDeploymentRoute(ctx context.Context, deploymentID string) error {
	m.removed = append(m.removed, deploymentID)
	if err := m.deploymentErrors[deploymentID]; err != nil {
		return err
	}
	return nil
}

func (m *mockRouteRemover) RemoveBranchRoute(ctx context.Context, projectID, branch string) error {
	m.removedBranches = append(m.removedBranches, projectID+":"+branch)
	return m.branchError
}

// --- Tests ---

func TestEventRouter_Ping(t *testing.T) {
	router := NewGitHubEventRouter(nil, nil, nil, slog.Default())
	err := router.Route(context.Background(), "ping", []byte("{}"), "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventRouter_Unknown(t *testing.T) {
	router := NewGitHubEventRouter(nil, nil, nil, slog.Default())
	err := router.Route(context.Background(), "unknown_event", []byte("{}"), "delivery-2")
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

	handler := NewPushHandler(projectRepo, deploySvc, &mockRouteRemover{}, slog.Default())

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

func TestPushHandler_BranchDeletionCleansPreviewRoutes(t *testing.T) {
	projectRepo := &mockProjectRepo{projects: map[string]*models.Project{
		"user/repo": {ID: "proj-1"},
	}}
	deploySvc := &mockDeploymentCreator{deactivated: map[string][]models.Deployment{
		"proj-1:feature/test": {{ID: "deploy-1"}, {ID: "deploy-2"}},
	}}
	routes := &mockRouteRemover{}
	handler := NewPushHandler(projectRepo, deploySvc, routes, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"ref":        "refs/heads/feature/test",
		"deleted":    true,
		"repository": map[string]string{"full_name": "user/repo"},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(routes.removed), 2; got != want {
		t.Fatalf("deployment route removals = %d, want %d", got, want)
	}
	if got, want := routes.removedBranches, []string{"proj-1:feature/test"}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("branch route removals = %v, want %v", got, want)
	}
}

func TestPushHandler_IgnoresTagPush(t *testing.T) {
	handler := NewPushHandler(nil, nil, nil, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"ref":     "refs/tags/v1.0",
		"after":   "abc123",
		"deleted": true,
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

	handler := NewPushHandler(projectRepo, deploySvc, &mockRouteRemover{}, slog.Default())

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

func TestPushThenPullRequestAssociatesExistingPreview(t *testing.T) {
	project := &models.Project{
		ID: "proj-1", ProductionBranch: "main", AutoDeploy: true, PreviewDeployments: true,
	}
	projects := &mockProjectRepo{projects: map[string]*models.Project{"user/repo": project}}
	deployments := &mockDeploymentCreator{commits: make(map[string]*models.Deployment)}
	push := NewPushHandler(projects, deployments, &mockRouteRemover{}, slog.Default())
	pr := NewPullRequestHandler(projects, deployments, &mockRouteRemover{}, slog.Default())

	pushPayload, _ := json.Marshal(map[string]any{
		"ref": "refs/heads/feature/test", "after": "def456",
		"repository":   map[string]string{"full_name": "user/repo"},
		"installation": map[string]int64{"id": 99},
		"head_commit":  map[string]any{"id": "def456", "message": "change", "author": map[string]string{"name": "Test"}},
	})
	if err := push.Handle(context.Background(), pushPayload, "push-first"); err != nil {
		t.Fatal(err)
	}
	prPayload, _ := json.Marshal(map[string]any{
		"action": "opened", "number": 42,
		"pull_request": map[string]any{
			"title": "My PR", "head": map[string]string{"ref": "feature/test", "sha": "def456"},
			"base": map[string]string{"ref": "main"},
		},
		"repository":   map[string]string{"full_name": "user/repo"},
		"installation": map[string]int64{"id": 99},
	})
	if err := pr.Handle(context.Background(), prPayload, "pr-second"); err != nil {
		t.Fatal(err)
	}

	if len(deployments.created) != 1 {
		t.Fatalf("deployments created = %d, want 1", len(deployments.created))
	}
	if len(deployments.associated) != 1 || deployments.associated[0] != 42 {
		t.Fatalf("PR associations = %v, want [42]", deployments.associated)
	}
	existing := deployments.commits["proj-1:feature/test:def456"]
	if existing.GitHubPRNumber == nil || *existing.GitHubPRNumber != 42 {
		t.Fatalf("existing deployment PR = %v, want 42", existing.GitHubPRNumber)
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
	if got, want := routes.removedBranches, []string{"proj-1:feature/test"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("branch route removals = %v, want %v", got, want)
	}
}

func TestPRHandler_ClosedReturnsRouteErrorsAfterAttemptingAllCleanup(t *testing.T) {
	deploymentRouteErr := errors.New("deployment route unavailable")
	branchRouteErr := errors.New("branch route unavailable")
	projectRepo := &mockProjectRepo{projects: map[string]*models.Project{
		"user/repo": {ID: "proj-1", PreviewDeployments: true},
	}}
	deploySvc := &mockDeploymentCreator{deactivated: map[string][]models.Deployment{
		"proj-1:feature/test": {{ID: "deploy-1"}, {ID: "deploy-2"}},
	}}
	routes := &mockRouteRemover{
		deploymentErrors: map[string]error{"deploy-1": deploymentRouteErr},
		branchError:      branchRouteErr,
	}
	handler := NewPullRequestHandler(projectRepo, deploySvc, routes, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "closed",
		"number": 42,
		"pull_request": map[string]interface{}{
			"head": map[string]string{"ref": "feature/test"},
		},
		"repository": map[string]string{"full_name": "user/repo"},
	})

	err := handler.Handle(context.Background(), payload, "delivery-1")
	if !errors.Is(err, deploymentRouteErr) || !errors.Is(err, branchRouteErr) {
		t.Fatalf("error = %v, want both cleanup errors", err)
	}
	if got := len(routes.removed); got != 2 {
		t.Errorf("deployment route attempts = %d, want 2", got)
	}
	if got := len(routes.removedBranches); got != 1 {
		t.Errorf("branch route attempts = %d, want 1", got)
	}
}

func TestPushHandler_BranchDeletionReturnsCleanupErrors(t *testing.T) {
	deactivateErr := errors.New("database unavailable")
	branchRouteErr := errors.New("caddy unavailable")
	projectRepo := &mockProjectRepo{projects: map[string]*models.Project{
		"user/repo": {ID: "proj-1"},
	}}
	deploySvc := &mockDeploymentCreator{deactivateErr: deactivateErr}
	routes := &mockRouteRemover{branchError: branchRouteErr}
	handler := NewPushHandler(projectRepo, deploySvc, routes, slog.Default())

	payload, _ := json.Marshal(map[string]interface{}{
		"ref":        "refs/heads/feature/test",
		"deleted":    true,
		"repository": map[string]string{"full_name": "user/repo"},
	})
	err := handler.Handle(context.Background(), payload, "delivery-1")
	if !errors.Is(err, deactivateErr) || !errors.Is(err, branchRouteErr) {
		t.Fatalf("error = %v, want both cleanup errors", err)
	}
	if got := len(routes.removedBranches); got != 1 {
		t.Errorf("branch route attempts = %d, want 1", got)
	}
}

func TestInstallationHandler_Deleted(t *testing.T) {
	projectRepo := &mockProjectRepo{
		projects: make(map[string]*models.Project),
	}

	handler := NewInstallationHandler(projectRepo, nil, slog.Default())

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

func TestPushHandler_FansOutToEveryProjectForSource(t *testing.T) {
	projectRepo := &mockProjectRepo{projectLists: map[string][]models.Project{
		"user/repo": {
			{ID: "proj-1", ProductionBranch: "main", AutoDeploy: true},
			{ID: "proj-2", ProductionBranch: "main", AutoDeploy: true},
		},
	}}
	deployments := &mockDeploymentCreator{commits: make(map[string]*models.Deployment)}
	handler := NewPushHandler(projectRepo, deployments, &mockRouteRemover{}, slog.Default())
	payload, _ := json.Marshal(map[string]any{
		"ref":          "refs/heads/main",
		"after":        "0123456789012345678901234567890123456789",
		"repository":   map[string]string{"full_name": "user/repo"},
		"installation": map[string]int64{"id": 99},
		"head_commit":  map[string]any{"message": "fan out", "author": map[string]string{"name": "Test"}},
	})

	if err := handler.Handle(context.Background(), payload, "delivery-fanout"); err != nil {
		t.Fatal(err)
	}
	if len(deployments.created) != 2 {
		t.Fatalf("deployments created = %d, want 2", len(deployments.created))
	}
	if deployments.created[0].ProjectID != "proj-1" || deployments.created[1].ProjectID != "proj-2" {
		t.Fatalf("project fan-out = %q, %q", deployments.created[0].ProjectID, deployments.created[1].ProjectID)
	}
}

func TestPullRequestHandler_RejectsForkPreviewPermanently(t *testing.T) {
	projectRepo := &mockProjectRepo{projects: map[string]*models.Project{
		"base/repo": {ID: "proj-1", PreviewDeployments: true},
	}}
	deployments := &mockDeploymentCreator{}
	handler := NewPullRequestHandler(projectRepo, deployments, &mockRouteRemover{}, slog.Default())
	payload, _ := json.Marshal(map[string]any{
		"action": "opened",
		"number": 7,
		"pull_request": map[string]any{
			"title": "Fork PR",
			"head": map[string]any{
				"ref": "feature", "sha": "abc",
				"repo": map[string]any{"full_name": "contributor/repo", "fork": true},
			},
			"base": map[string]string{"ref": "main"},
		},
		"repository":   map[string]string{"full_name": "base/repo"},
		"installation": map[string]int64{"id": 99},
	})

	err := handler.Handle(context.Background(), payload, "delivery-fork")
	var permanentErr *PermanentWebhookError
	if !errors.As(err, &permanentErr) {
		t.Fatalf("error = %v, want PermanentWebhookError", err)
	}
	if len(deployments.created) != 0 {
		t.Fatalf("deployments created = %d, want 0", len(deployments.created))
	}
}

func TestInstallationHandler_TracksSuspensionAndRepositoryAccess(t *testing.T) {
	projectRepo := &mockProjectRepo{}
	tokens := &mockTokenInvalidator{}
	handler := NewInstallationHandler(projectRepo, tokens, slog.Default())

	suspendPayload, _ := json.Marshal(map[string]any{
		"action":       "suspend",
		"installation": map[string]any{"id": 99, "account": map[string]string{"login": "octo"}},
	})
	if err := handler.Handle(context.Background(), suspendPayload, "suspend"); err != nil {
		t.Fatal(err)
	}
	if got := projectRepo.statuses[99]; got != models.GitHubConnectionSuspended {
		t.Fatalf("suspended status = %q", got)
	}

	repositoriesPayload, _ := json.Marshal(map[string]any{
		"action":       "removed",
		"installation": map[string]int64{"id": 99},
		"repositories_added": []any{
			map[string]any{"id": 1, "full_name": "octo/added"},
		},
		"repositories_removed": []any{
			map[string]any{"id": 2, "full_name": "octo/removed"},
		},
	})
	if err := handler.HandleRepositories(context.Background(), repositoriesPayload, "access"); err != nil {
		t.Fatal(err)
	}
	if len(projectRepo.accessChanges) != 2 ||
		!projectRepo.accessChanges[0].granted ||
		projectRepo.accessChanges[1].granted {
		t.Fatalf("access changes = %#v", projectRepo.accessChanges)
	}
	if got := tokens.installationIDs; len(got) != 2 || got[0] != 99 || got[1] != 99 {
		t.Fatalf("token invalidations = %v, want [99 99]", got)
	}
}

func TestInstallationHandler_FollowsRepositoryAndOwnerRenames(t *testing.T) {
	projectRepo := &mockProjectRepo{}
	handler := NewInstallationHandler(projectRepo, nil, slog.Default())

	repositoryPayload, _ := json.Marshal(map[string]any{
		"action": "renamed",
		"repository": map[string]any{
			"id": 123, "name": "new-name", "full_name": "octo/new-name",
			"owner": map[string]string{"login": "octo"},
		},
		"installation": map[string]int64{"id": 99},
		"changes": map[string]any{
			"repository": map[string]any{"name": map[string]string{"from": "old-name"}},
		},
	})
	if err := handler.HandleRepository(context.Background(), repositoryPayload, "rename-repo"); err != nil {
		t.Fatal(err)
	}
	if got := projectRepo.identityChanges; len(got) != 1 ||
		got[0].oldFullName != "octo/old-name" ||
		got[0].newFullName != "octo/new-name" ||
		got[0].repositoryID != 123 {
		t.Fatalf("identity changes = %#v", got)
	}

	targetPayload, _ := json.Marshal(map[string]any{
		"action":       "renamed",
		"installation": map[string]int64{"id": 99},
		"account":      map[string]string{"login": "new-owner"},
		"changes":      map[string]any{"login": map[string]string{"from": "old-owner"}},
	})
	if err := handler.HandleTarget(context.Background(), targetPayload, "rename-owner"); err != nil {
		t.Fatal(err)
	}
	if got := projectRepo.installationMoves; len(got) != 1 ||
		got[0].oldOwner != "old-owner" || got[0].newOwner != "new-owner" {
		t.Fatalf("owner changes = %#v", got)
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
