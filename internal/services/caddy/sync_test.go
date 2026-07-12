package caddy

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockDeployRepo struct {
	deployments []ActiveDeployment
	err         error
	started     chan struct{}
	release     chan struct{}
}

func (m *mockDeployRepo) ListActiveWithProject(ctx context.Context) ([]ActiveDeployment, error) {
	if m.started != nil {
		close(m.started)
	}
	if m.release != nil {
		select {
		case <-m.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return m.deployments, m.err
}

type mockDomainRepo struct {
	domains []VerifiedDomain
	err     error
}

func (m *mockDomainRepo) ListVerifiedWithProject(ctx context.Context) ([]VerifiedDomain, error) {
	return m.domains, m.err
}

func TestSyncService_SyncAll(t *testing.T) {
	var receivedConfig CaddyConfig
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/load" {
			json.NewDecoder(r.Body).Decode(&receivedConfig)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	builder := newTestBuilder()

	deploys := &mockDeployRepo{
		deployments: []ActiveDeployment{
			{
				DeploymentID: "dpl_001",
				ProjectID:    "prj_001",
				ProjectSlug:  "my-app",
				Branch:       "main",
				IsProduction: true,
				ArtifactPath: "/app/out",
				Framework:    "vite",
			},
		},
	}
	domains := &mockDomainRepo{
		domains: []VerifiedDomain{
			{
				DomainID:           "dom_001",
				Domain:             "myapp.com",
				ProjectID:          "prj_001",
				ProductionArtifact: "/app/out",
				Framework:          "vite",
			},
		},
	}

	svc := NewSyncService(client, builder, deploys, domains, slog.Default())
	err := svc.SyncAll(context.Background())
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// Verify config was loaded
	if receivedConfig.Admin == nil {
		t.Fatal("no config received by Caddy")
	}
	routes := receivedConfig.Apps.HTTP.Servers["main"].Routes
	if len(routes) < 3 { // platform + domain + production + preview + branch
		t.Errorf("expected at least 3 routes, got %d", len(routes))
	}
}

func TestSyncService_SerializesSnapshotLoadWithIncrementalMutation(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	builder := newTestBuilder()
	started := make(chan struct{})
	release := make(chan struct{})
	svc := NewSyncService(client, builder, &mockDeployRepo{started: started, release: release}, &mockDomainRepo{}, slog.Default())
	mgr := NewRouteManager(client, builder, slog.Default())

	syncDone := make(chan error, 1)
	go func() { syncDone <- svc.SyncAll(context.Background()) }()
	<-started

	mutationDone := make(chan error, 1)
	go func() {
		mutationDone <- mgr.AddDeploymentRoute(context.Background(), ActiveDeployment{
			DeploymentID: "dpl_new", ProjectID: "prj_1", ProjectSlug: "app", Branch: "main", ArtifactPath: "/tmp/out",
		})
	}()

	select {
	case request := <-requests:
		t.Fatalf("incremental mutation escaped while full sync held snapshot lock: %s", request)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if err := <-syncDone; err != nil {
		t.Fatalf("SyncAll: %v", err)
	}
	if err := <-mutationDone; err != nil {
		t.Fatalf("AddDeploymentRoute: %v", err)
	}

	if first := <-requests; first != "POST /load" {
		t.Fatalf("first request = %q, want full config load", first)
	}
	if second := <-requests; second != "POST /config/apps/http/servers/main/routes" {
		t.Fatalf("second request = %q, want incremental route after load", second)
	}
}

func TestSyncService_WaitForCaddy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	builder := newTestBuilder()
	svc := NewSyncService(client, builder, &mockDeployRepo{}, &mockDomainRepo{}, slog.Default())

	err := svc.WaitForCaddy(context.Background(), 5*time.Second)
	if err != nil {
		t.Fatalf("WaitForCaddy failed: %v", err)
	}
}

func TestSyncService_WaitForCaddy_Timeout(t *testing.T) {
	client := NewCaddyClient("http://127.0.0.1:1", slog.Default())
	builder := newTestBuilder()
	svc := NewSyncService(client, builder, &mockDeployRepo{}, &mockDomainRepo{}, slog.Default())

	err := svc.WaitForCaddy(context.Background(), 1*time.Second)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
