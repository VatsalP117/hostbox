package caddy

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestRouteManager_AddDeploymentRoute(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	builder := newTestBuilder()
	mgr := NewRouteManager(client, builder, slog.Default())

	err := mgr.AddDeploymentRoute(context.Background(), ActiveDeployment{
		DeploymentID: "dpl_test123",
		ProjectID:    "prj_001",
		ProjectSlug:  "my-app",
		Branch:       "main",
		ArtifactPath: "/app/deployments/prj_001/dpl_test123",
		Framework:    "vite",
	})
	if err != nil {
		t.Fatalf("AddDeploymentRoute failed: %v", err)
	}
	if receivedPath != "/config/apps/http/servers/main/routes" {
		t.Errorf("unexpected path: %s", receivedPath)
	}
}

func TestRouteManager_UpdateProductionRoute(t *testing.T) {
	var paths []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	builder := newTestBuilder()
	mgr := NewRouteManager(client, builder, slog.Default())

	err := mgr.UpdateProductionRoute(context.Background(), "my-app", "prj_001", "/app/out", "vite")
	if err != nil {
		t.Fatalf("UpdateProductionRoute failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// Should have DELETE (old) + POST (new)
	if len(paths) != 2 {
		t.Fatalf("expected 2 requests, got %d: %v", len(paths), paths)
	}
	if !strings.HasPrefix(paths[0], "DELETE") {
		t.Errorf("first request should be DELETE, got %s", paths[0])
	}
	if !strings.HasPrefix(paths[1], "POST") {
		t.Errorf("second request should be POST, got %s", paths[1])
	}
}

func TestRouteManager_RemoveAllProjectRoutes(t *testing.T) {
	var paths []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	builder := newTestBuilder()
	mgr := NewRouteManager(client, builder, slog.Default())

	err := mgr.RemoveAllProjectRoutes(context.Background(), "prj_001",
		[]string{"dpl_1", "dpl_2"},
		[]string{"main", "develop"},
		[]string{"dom_1"},
	)
	if err != nil {
		t.Fatalf("RemoveAllProjectRoutes failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// 1 production + 2 branches + 2 deployments + 1 domain = 6 DELETEs
	if len(paths) != 6 {
		t.Fatalf("expected 6 DELETE requests, got %d: %v", len(paths), paths)
	}
}
