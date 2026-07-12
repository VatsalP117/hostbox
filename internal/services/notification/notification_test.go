package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/logger"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/util"
	"github.com/VatsalP117/hostbox/migrations"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func setupNotifTest(t *testing.T) (*repository.NotificationRepository, *repository.ProjectRepository, *sql.DB) {
	t.Helper()
	db := setupTestDB(t)
	repos := repository.New(db)

	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec(`INSERT INTO users (id, email, password_hash, display_name, is_admin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"user-1", "test@example.com", "hash", "Test User", 1, now, now)

	return repos.Notification, repos.Project, db
}

type recordingNotificationClient struct {
	calls chan string
}

func (c *recordingNotificationClient) Send(_ context.Context, webhookURL string, _ NotificationPayload) error {
	c.calls <- webhookURL
	return nil
}

func TestDiscordClient_Send(t *testing.T) {
	var mu sync.Mutex
	var received map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &DiscordClient{httpClient: srv.Client()}

	payload := NotificationPayload{
		Event:   EventDeploySuccess,
		Project: ProjectInfo{ID: "prj-1", Name: "My App", Slug: "my-app"},
		Deployment: &DeploymentInfo{
			ID:              "dpl-1",
			Status:          "ready",
			Branch:          "main",
			CommitSHA:       "abc1234",
			CommitMessage:   "fix: login bug",
			BuildDurationMs: 45000,
			DeploymentURL:   "https://my-app.example.com",
		},
		Timestamp: "2024-01-15T10:30:46Z",
		ServerURL: "https://hostbox.example.com",
	}

	err := client.Send(context.Background(), srv.URL, payload)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	embeds, ok := received["embeds"].([]any)
	if !ok || len(embeds) == 0 {
		t.Fatal("expected embeds in discord payload")
	}

	embed := embeds[0].(map[string]any)
	if embed["color"] != float64(colorGreen) {
		t.Errorf("expected green color, got %v", embed["color"])
	}
}

func TestSlackClient_Send(t *testing.T) {
	var mu sync.Mutex
	var received map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &SlackClient{httpClient: srv.Client()}

	payload := NotificationPayload{
		Event:   EventDeployFailure,
		Project: ProjectInfo{ID: "prj-1", Name: "My App", Slug: "my-app"},
		Deployment: &DeploymentInfo{
			ID:           "dpl-1",
			Status:       "failed",
			Branch:       "feat/broken",
			CommitSHA:    "def5678",
			ErrorMessage: "Module not found: react-dom",
			DashboardURL: "https://hostbox.example.com/projects/my-app/deployments/dpl-1",
		},
		Timestamp: "2024-01-15T10:30:46Z",
		ServerURL: "https://hostbox.example.com",
	}

	err := client.Send(context.Background(), srv.URL, payload)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	blocks, ok := received["blocks"].([]any)
	if !ok || len(blocks) == 0 {
		t.Fatal("expected blocks in slack payload")
	}

	header := blocks[0].(map[string]any)
	if header["type"] != "header" {
		t.Errorf("expected header block type, got %v", header["type"])
	}
}

func TestWebhookClient_Send(t *testing.T) {
	var mu sync.Mutex
	var received NotificationPayload
	var headers http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		headers = r.Header.Clone()
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &WebhookClient{httpClient: srv.Client()}

	payload := NotificationPayload{
		Event:   EventDomainVerified,
		Project: ProjectInfo{ID: "prj-1", Name: "My App", Slug: "my-app"},
		Domain:  &DomainInfo{ID: "dom-1", Domain: "myapp.com", Verified: true},
	}

	err := client.Send(context.Background(), srv.URL, payload)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Event != EventDomainVerified {
		t.Errorf("expected event %s, got %s", EventDomainVerified, received.Event)
	}
	if headers.Get("X-Hostbox-Event") != EventDomainVerified {
		t.Errorf("expected X-Hostbox-Event header")
	}
	if headers.Get("User-Agent") != "Hostbox-Webhook/1.0" {
		t.Errorf("expected Hostbox-Webhook/1.0 user agent")
	}
}

func TestService_Dispatch(t *testing.T) {
	notifRepo, projectRepo, _ := setupNotifTest(t)

	// Create a project
	p := &models.Project{Name: "Test App", Slug: "test-app", OwnerID: "user-1"}
	projectRepo.Create(context.Background(), p)

	const webhookURL = "https://198.51.100.10/hook"

	// Create notification configs: one project-level, one global
	notifRepo.Create(context.Background(), &models.NotificationConfig{
		ID:         util.NewID(),
		ProjectID:  &p.ID,
		Channel:    "webhook",
		WebhookURL: webhookURL,
		Events:     "all",
		Enabled:    true,
	})
	notifRepo.Create(context.Background(), &models.NotificationConfig{
		ID:         util.NewID(),
		ProjectID:  nil,
		Channel:    "webhook",
		WebhookURL: webhookURL,
		Events:     "deploy_success,deploy_failure",
		Enabled:    true,
	})

	l := logger.Setup("error", "text")
	svc := NewService(notifRepo, l)
	calls := make(chan string, 2)
	svc.clients["webhook"] = &recordingNotificationClient{calls: calls}

	payload := NotificationPayload{
		Project: ProjectInfo{ID: p.ID, Name: p.Name, Slug: p.Slug},
		Deployment: &DeploymentInfo{
			ID:     "dpl-1",
			Status: "ready",
			Branch: "main",
		},
	}

	svc.Dispatch(context.Background(), EventDeploySuccess, payload)

	for i := 0; i < 2; i++ {
		select {
		case got := <-calls:
			if got != webhookURL {
				t.Errorf("webhook URL = %q, want %q", got, webhookURL)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for project and global notification sends")
		}
	}
}

func TestService_Dispatch_RejectsUnsafeStoredURL(t *testing.T) {
	notifRepo, projectRepo, _ := setupNotifTest(t)
	p := &models.Project{Name: "Test App", Slug: "test-app", OwnerID: "user-1"}
	if err := projectRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := notifRepo.Create(context.Background(), &models.NotificationConfig{
		ID:         util.NewID(),
		ProjectID:  &p.ID,
		Channel:    "webhook",
		WebhookURL: "https://127.0.0.1/hook",
		Events:     "all",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("create notification: %v", err)
	}

	svc := NewService(notifRepo, logger.Setup("error", "text"))
	calls := make(chan string, 1)
	svc.clients["webhook"] = &recordingNotificationClient{calls: calls}
	svc.Dispatch(context.Background(), EventDeploySuccess, NotificationPayload{
		Project: ProjectInfo{ID: p.ID, Name: p.Name, Slug: p.Slug},
	})

	select {
	case got := <-calls:
		t.Fatalf("unsafe webhook reached client: %q", got)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestService_SendTest_ValidatesWebhookURL(t *testing.T) {
	notifRepo, _, _ := setupNotifTest(t)
	svc := NewService(notifRepo, logger.Setup("error", "text"))
	calls := make(chan string, 1)
	svc.clients["webhook"] = &recordingNotificationClient{calls: calls}

	unsafeConfig := &models.NotificationConfig{Channel: "webhook", WebhookURL: "http://198.51.100.10/hook"}
	if err := svc.SendTest(context.Background(), unsafeConfig, NotificationPayload{}); err == nil {
		t.Fatal("SendTest accepted an insecure HTTP URL")
	}
	select {
	case got := <-calls:
		t.Fatalf("unsafe webhook reached client: %q", got)
	default:
	}

	const validURL = "https://198.51.100.10/hook"
	validConfig := &models.NotificationConfig{Channel: "webhook", WebhookURL: validURL}
	if err := svc.SendTest(context.Background(), validConfig, NotificationPayload{}); err != nil {
		t.Fatalf("SendTest rejected valid HTTPS URL: %v", err)
	}
	select {
	case got := <-calls:
		if got != validURL {
			t.Fatalf("webhook URL = %q, want %q", got, validURL)
		}
	default:
		t.Fatal("valid HTTPS webhook did not reach client")
	}
}

func TestService_Dispatch_DisabledConfig(t *testing.T) {
	notifRepo, projectRepo, _ := setupNotifTest(t)

	p := &models.Project{Name: "Test App", Slug: "test-app", OwnerID: "user-1"}
	projectRepo.Create(context.Background(), p)

	var mu sync.Mutex
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Create disabled config
	notifRepo.Create(context.Background(), &models.NotificationConfig{
		ID:         util.NewID(),
		ProjectID:  &p.ID,
		Channel:    "webhook",
		WebhookURL: srv.URL,
		Events:     "all",
		Enabled:    false,
	})

	l := logger.Setup("error", "text")
	svc := NewService(notifRepo, l)
	svc.clients["webhook"] = &WebhookClient{httpClient: srv.Client()}

	payload := NotificationPayload{
		Project: ProjectInfo{ID: p.ID, Name: p.Name, Slug: p.Slug},
	}

	svc.Dispatch(context.Background(), EventDeploySuccess, payload)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if calls != 0 {
		t.Errorf("expected 0 calls for disabled config, got %d", calls)
	}
	mu.Unlock()
}

func TestFindByProjectAndEvent(t *testing.T) {
	notifRepo, projectRepo, _ := setupNotifTest(t)

	p := &models.Project{Name: "Test App", Slug: "test-app", OwnerID: "user-1"}
	projectRepo.Create(context.Background(), p)

	// Config matching deploy_success
	notifRepo.Create(context.Background(), &models.NotificationConfig{
		ID:         util.NewID(),
		ProjectID:  &p.ID,
		Channel:    "discord",
		WebhookURL: "https://discord.com/api/webhooks/123",
		Events:     "deploy_success,deploy_failure",
		Enabled:    true,
	})

	// Config only for domain events
	notifRepo.Create(context.Background(), &models.NotificationConfig{
		ID:         util.NewID(),
		ProjectID:  &p.ID,
		Channel:    "slack",
		WebhookURL: "https://hooks.slack.com/services/123",
		Events:     "domain_verified",
		Enabled:    true,
	})

	configs, err := notifRepo.FindByProjectAndEvent(context.Background(), p.ID, EventDeploySuccess)
	if err != nil {
		t.Fatalf("FindByProjectAndEvent failed: %v", err)
	}

	if len(configs) != 1 {
		t.Fatalf("expected 1 matching config, got %d", len(configs))
	}
	if configs[0].Channel != "discord" {
		t.Errorf("expected discord channel, got %s", configs[0].Channel)
	}
}
