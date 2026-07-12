package caddy

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	deploysvc "github.com/VatsalP117/hostbox/internal/services/deployment"
)

func TestProductionActivatorAdapter_ActivatesProductionRoute(t *testing.T) {
	var addedRoute CaddyRoute
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			if r.URL.Path != "/id/route-prod-prj_001" {
				t.Errorf("delete path = %q", r.URL.Path)
			}
		case http.MethodPost:
			if r.URL.Path != "/config/apps/http/servers/main/routes" {
				t.Errorf("post path = %q", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&addedRoute); err != nil {
				t.Errorf("decode route: %v", err)
			}
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewRouteManager(NewCaddyClient(server.URL, slog.Default()), newTestBuilder(), slog.Default())
	adapter := &ProductionActivatorAdapter{Manager: manager}
	activation := deploysvc.ProductionActivation{
		ProjectID:    "prj_001",
		ProjectSlug:  "my-app",
		ArtifactPath: "/app/deployments/prj_001/dpl_001",
		Framework:    "vite",
	}

	if err := adapter.ActivateProduction(context.Background(), activation); err != nil {
		t.Fatalf("ActivateProduction: %v", err)
	}

	if addedRoute.ID != "route-prod-prj_001" {
		t.Errorf("route ID = %q", addedRoute.ID)
	}
	if len(addedRoute.Match) != 1 || len(addedRoute.Match[0].Host) != 1 || addedRoute.Match[0].Host[0] != "my-app.example.com" {
		t.Errorf("unexpected route host: %+v", addedRoute.Match)
	}
	if !routeContainsFileServerRoot(addedRoute, activation.ArtifactPath) {
		t.Errorf("route does not serve artifact %q: %+v", activation.ArtifactPath, addedRoute)
	}
}

func TestProductionActivatorAdapter_ReturnsRouteUpdateFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "route rejected", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewRouteManager(NewCaddyClient(server.URL, slog.Default()), newTestBuilder(), slog.Default())
	adapter := &ProductionActivatorAdapter{Manager: manager}
	err := adapter.ActivateProduction(context.Background(), deploysvc.ProductionActivation{
		ProjectID:    "prj_001",
		ProjectSlug:  "my-app",
		ArtifactPath: "/app/deployments/prj_001/dpl_001",
		Framework:    "vite",
	})
	if err == nil {
		t.Fatal("expected route update error")
	}
}

func routeContainsFileServerRoot(route CaddyRoute, root string) bool {
	for _, handler := range route.Handle {
		if handler.Handler == "file_server" && handler.Root == root {
			return true
		}
		for _, nestedRoute := range handler.Routes {
			if routeContainsFileServerRoot(nestedRoute, root) {
				return true
			}
		}
	}
	return false
}
