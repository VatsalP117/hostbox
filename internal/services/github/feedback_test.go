package github

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/migrations"
)

type feedbackStatusCall struct {
	installationID int64
	owner          string
	repo           string
	deploymentID   int64
	request        CreateDeploymentStatusRequest
}

type fakeFeedbackClient struct {
	createDeploymentCalls int
	statusCalls           []feedbackStatusCall
	comments              []IssueComment
	createCommentCalls    int
	updateCommentCalls    int
	lastCommentBody       string
	statusFailures        int
}

func (f *fakeFeedbackClient) CreateDeployment(_ context.Context, installationID int64, owner, repo string, req CreateDeploymentRequest) (*DeploymentResponse, error) {
	f.createDeploymentCalls++
	if installationID != 99 || owner != "octo" || repo != "app" || req.Ref == "" {
		return nil, errors.New("unexpected create deployment metadata")
	}
	return &DeploymentResponse{ID: 1234}, nil
}

func (f *fakeFeedbackClient) CreateDeploymentStatus(_ context.Context, installationID int64, owner, repo string, deploymentID int64, req CreateDeploymentStatusRequest) error {
	f.statusCalls = append(f.statusCalls, feedbackStatusCall{installationID, owner, repo, deploymentID, req})
	if f.statusFailures > 0 {
		f.statusFailures--
		return errors.New("temporary status failure")
	}
	return nil
}

func (f *fakeFeedbackClient) ListPRComments(context.Context, int64, string, string, int) ([]IssueComment, error) {
	return append([]IssueComment(nil), f.comments...), nil
}

func TestLifecycleReporterRetriesTransientStatusFailureWithoutCreatingAnotherDeployment(t *testing.T) {
	client := &fakeFeedbackClient{statusFailures: 1}
	store := &fakeGitHubDeploymentStore{ids: make(map[string]int64)}
	reporter, err := NewLifecycleReporter(
		FeedbackClientProviderFunc(func() (FeedbackClient, error) { return client, nil }),
		store,
		"https://hostbox.example.com",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	repo := "octo/app"
	installationID := int64(99)
	project := &models.Project{ID: "p1", Name: "App", GitHubRepo: &repo, GitHubInstallationID: &installationID}
	deployment := &models.Deployment{ID: "d1", CommitSHA: strings.Repeat("a", 40), Branch: "main", Status: models.DeploymentStatusReady}

	if err := reporter.Report(context.Background(), project, deployment); err != nil {
		t.Fatal(err)
	}
	if client.createDeploymentCalls != 1 {
		t.Fatalf("github deployments created = %d, want 1", client.createDeploymentCalls)
	}
	if len(client.statusCalls) != 2 {
		t.Fatalf("status attempts = %d, want 2", len(client.statusCalls))
	}
	if deployment.GitHubDeployID == nil || *deployment.GitHubDeployID != 1234 {
		t.Fatalf("persisted deployment ID = %v", deployment.GitHubDeployID)
	}
}

func TestLifecycleReporterReloadInsideLockPreventsReadyRegression(t *testing.T) {
	db, err := database.Open(t.TempDir() + "/feedback.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	user := &models.User{Email: "feedback@test.local", PasswordHash: "hash"}
	if err := repository.NewUserRepository(db).Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	repoName := "octo/app"
	installationID := int64(99)
	project := &models.Project{
		OwnerID: user.ID, Name: "App", Slug: "app", ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
		GitHubRepo: &repoName, GitHubInstallationID: &installationID,
	}
	if err := repository.NewProjectRepository(db).Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	prNumber := 42
	stored := &models.Deployment{
		ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40), Branch: "feature/test",
		Status: models.DeploymentStatusBuilding, GitHubPRNumber: &prNumber,
	}
	if err := deploymentRepo.Create(ctx, stored); err != nil {
		t.Fatal(err)
	}
	staleBuilding := *stored
	if err := deploymentRepo.UpdateStatus(ctx, stored.ID, models.DeploymentStatusReady, nil); err != nil {
		t.Fatal(err)
	}
	currentReady, err := deploymentRepo.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatal(err)
	}

	client := &fakeFeedbackClient{}
	reporter, err := NewLifecycleReporter(
		FeedbackClientProviderFunc(func() (FeedbackClient, error) { return client, nil }),
		deploymentRepo,
		"https://hostbox.example.com",
		nil,
		"preview.example.com",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := reporter.Report(ctx, project, currentReady); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Report(ctx, project, &staleBuilding); err != nil {
		t.Fatal(err)
	}
	if len(client.statusCalls) != 2 || client.statusCalls[1].request.State != "success" {
		t.Fatalf("status calls = %#v, final state must remain success", client.statusCalls)
	}
	if !strings.Contains(client.lastCommentBody, "Preview Deployment Ready") {
		t.Fatalf("final comment regressed: %q", client.lastCommentBody)
	}
	if staleBuilding.Status != models.DeploymentStatusReady || staleBuilding.GitHubPRNumber == nil {
		t.Fatalf("stale caller was not refreshed: %+v", staleBuilding)
	}
}

func (f *fakeFeedbackClient) CreatePRComment(_ context.Context, _ int64, _, _ string, _ int, body string) (*IssueComment, error) {
	f.createCommentCalls++
	f.lastCommentBody = body
	comment := IssueComment{ID: 77, Body: body}
	f.comments = append(f.comments, comment)
	return &comment, nil
}

func (f *fakeFeedbackClient) UpdateComment(_ context.Context, _ int64, _, _ string, _ int64, body string) error {
	f.updateCommentCalls++
	f.lastCommentBody = body
	for i := range f.comments {
		if strings.Contains(f.comments[i].Body, commentMarker) {
			f.comments[i].Body = body
		}
	}
	return nil
}

type fakeGitHubDeploymentStore struct {
	ids   map[string]int64
	calls int
}

func (s *fakeGitHubDeploymentStore) SetGitHubDeployIDIfUnset(_ context.Context, deploymentID string, githubDeployID int64) (int64, error) {
	s.calls++
	if existing := s.ids[deploymentID]; existing > 0 {
		return existing, nil
	}
	s.ids[deploymentID] = githubDeployID
	return githubDeployID, nil
}

func TestLifecycleReporterCreatesOnceAndUpdatesStatusesAndMarkerComment(t *testing.T) {
	client := &fakeFeedbackClient{}
	store := &fakeGitHubDeploymentStore{ids: make(map[string]int64)}
	reporter, err := NewLifecycleReporter(
		FeedbackClientProviderFunc(func() (FeedbackClient, error) { return client, nil }),
		store,
		"https://hostbox.example.com",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"preview.example.com",
	)
	if err != nil {
		t.Fatal(err)
	}
	repo := "octo/app"
	installationID := int64(99)
	prNumber := 42
	commitMessage := "feat: preview\n\nmore detail"
	project := &models.Project{
		ID: "project-1", Name: "Web App", Slug: "web-app",
		GitHubRepo: &repo, GitHubInstallationID: &installationID,
	}
	deployment := &models.Deployment{
		ID: "deployment-1", ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40),
		CommitMessage: &commitMessage, Branch: "feature/preview",
		Status: models.DeploymentStatusQueued, GitHubPRNumber: &prNumber,
	}

	if err := reporter.Report(context.Background(), project, deployment); err != nil {
		t.Fatalf("report queued: %v", err)
	}
	previewURL := "https://web-app-deployment-1.example.com"
	duration := int64(45250)
	deployment.Status = models.DeploymentStatusReady
	deployment.DeploymentURL = &previewURL
	deployment.BuildDurationMs = &duration
	if err := reporter.Report(context.Background(), project, deployment); err != nil {
		t.Fatalf("report ready: %v", err)
	}

	if client.createDeploymentCalls != 1 {
		t.Fatalf("github deployments created = %d, want 1", client.createDeploymentCalls)
	}
	if store.calls != 1 || store.ids[deployment.ID] != 1234 {
		t.Fatalf("persist calls/ID = %d/%d, want 1/1234", store.calls, store.ids[deployment.ID])
	}
	if deployment.GitHubDeployID == nil || *deployment.GitHubDeployID != 1234 {
		t.Fatalf("deployment github ID = %v, want 1234", deployment.GitHubDeployID)
	}
	if len(client.statusCalls) != 2 {
		t.Fatalf("status calls = %d, want 2", len(client.statusCalls))
	}
	if client.statusCalls[0].request.State != "pending" || client.statusCalls[1].request.State != "success" {
		t.Fatalf("states = %q, %q", client.statusCalls[0].request.State, client.statusCalls[1].request.State)
	}
	if client.statusCalls[1].deploymentID != 1234 {
		t.Fatalf("updated deployment ID = %d, want 1234", client.statusCalls[1].deploymentID)
	}
	wantDashboardURL := "https://hostbox.example.com/projects/project-1/deployments/deployment-1"
	if client.statusCalls[1].request.LogURL != wantDashboardURL {
		t.Fatalf("log URL = %q, want %q", client.statusCalls[1].request.LogURL, wantDashboardURL)
	}
	if client.statusCalls[1].request.EnvironmentURL != previewURL {
		t.Fatalf("environment URL = %q, want %q", client.statusCalls[1].request.EnvironmentURL, previewURL)
	}
	if client.createCommentCalls != 1 || client.updateCommentCalls != 1 {
		t.Fatalf("comment create/update = %d/%d, want 1/1", client.createCommentCalls, client.updateCommentCalls)
	}
	if strings.Count(client.lastCommentBody, commentMarker) != 1 || !strings.Contains(client.lastCommentBody, "Preview Deployment Ready") {
		t.Fatalf("final comment did not update the Hostbox marker: %q", client.lastCommentBody)
	}
	if !strings.Contains(client.lastCommentBody, "45.25s") {
		t.Fatalf("final comment missing duration: %q", client.lastCommentBody)
	}
	if !strings.Contains(client.lastCommentBody, "https://web-app-feature-preview.preview.example.com") {
		t.Fatalf("final comment missing branch-stable URL: %q", client.lastCommentBody)
	}
}

func TestLifecycleReporterSkipsDisconnectedProjectWithoutRequestingClient(t *testing.T) {
	providerCalls := 0
	reporter, err := NewLifecycleReporter(
		FeedbackClientProviderFunc(func() (FeedbackClient, error) {
			providerCalls++
			return nil, errors.New("not configured")
		}),
		&fakeGitHubDeploymentStore{ids: make(map[string]int64)},
		"http://localhost:8080",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = reporter.Report(context.Background(), &models.Project{ID: "manual"}, &models.Deployment{ID: "d1"})
	if err != nil {
		t.Fatalf("disconnected project should be ignored: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("provider called %d times, want 0", providerCalls)
	}
}

func TestLifecycleReporterRejectsUnsafeOrIncompleteMetadata(t *testing.T) {
	client := &fakeFeedbackClient{}
	reporter, err := NewLifecycleReporter(
		FeedbackClientProviderFunc(func() (FeedbackClient, error) { return client, nil }),
		&fakeGitHubDeploymentStore{ids: make(map[string]int64)},
		"https://hostbox.example.com",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	badRepo := "octo/app/extra"
	installationID := int64(99)
	deployment := &models.Deployment{ID: "d1", CommitSHA: "abc", Status: models.DeploymentStatusQueued}
	if err := reporter.Report(context.Background(), &models.Project{GitHubRepo: &badRepo, GitHubInstallationID: &installationID}, deployment); err == nil {
		t.Fatal("expected malformed repository error")
	}
	validRepo := "octo/app"
	if err := reporter.Report(context.Background(), &models.Project{GitHubRepo: &validRepo}, deployment); err == nil {
		t.Fatal("expected missing installation error")
	}
	if client.createDeploymentCalls != 0 {
		t.Fatalf("created %d deployments for invalid metadata", client.createDeploymentCalls)
	}
}

func TestLifecycleFeedbackMapsEveryHostboxState(t *testing.T) {
	tests := map[string]string{
		"queued":    "pending",
		"building":  "in_progress",
		"ready":     "success",
		"failed":    "failure",
		"cancelled": "error",
	}
	for hostboxStatus, githubStatus := range tests {
		t.Run(hostboxStatus, func(t *testing.T) {
			if got := mapStatus(hostboxStatus); got != githubStatus {
				t.Fatalf("mapStatus(%q) = %q, want %q", hostboxStatus, got, githubStatus)
			}
			manager := &PRCommentManager{dashboardDomain: "hostbox.example.com"}
			body := manager.buildCommentBody(DeploymentInfo{
				ProjectName: "Web App",
				CommitSHA:   "abcdef123456",
				Status:      hostboxStatus,
			})
			if !strings.Contains(body, commentMarker) {
				t.Fatal("comment is missing Hostbox marker")
			}
			if !strings.Contains(strings.ToLower(body), hostboxStatus) {
				t.Fatalf("comment does not describe %q state: %q", hostboxStatus, body)
			}
		})
	}
}
