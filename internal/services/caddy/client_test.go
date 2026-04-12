package caddy

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCaddyClient_LoadConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/load" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	err := client.LoadConfig(context.Background(), &CaddyConfig{
		Admin: &CaddyAdmin{Listen: "localhost:2019"},
	})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
}

func TestCaddyClient_GetConfig(t *testing.T) {
	cfg := &CaddyConfig{
		Admin: &CaddyAdmin{Listen: "localhost:2019"},
		Apps:  &CaddyApps{HTTP: &CaddyHTTPApp{Servers: map[string]*CaddyServer{}}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	result, err := client.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if result.Admin == nil || result.Admin.Listen != "localhost:2019" {
		t.Error("unexpected config returned")
	}
}

func TestCaddyClient_AddRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/apps/http/servers/main/routes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	err := client.AddRoute(context.Background(), "main", CaddyRoute{
		ID:    "test-route",
		Match: []CaddyMatch{{Host: []string{"example.com"}}},
	})
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}
}

func TestCaddyClient_DeleteRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/id/route-test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	err := client.DeleteRoute(context.Background(), "route-test")
	if err != nil {
		t.Fatalf("DeleteRoute failed: %v", err)
	}
}

func TestCaddyClient_RetryOn5xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	err := client.LoadConfig(context.Background(), &CaddyConfig{})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestCaddyClient_NoRetryOn4xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	err := client.LoadConfig(context.Background(), &CaddyConfig{})
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestCaddyClient_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCaddyClient(server.URL, slog.Default())
	if !client.Healthy(context.Background()) {
		t.Error("expected healthy=true")
	}
}

func TestCaddyClient_Healthy_Unreachable(t *testing.T) {
	client := NewCaddyClient("http://127.0.0.1:1", slog.Default())
	if client.Healthy(context.Background()) {
		t.Error("expected healthy=false for unreachable")
	}
}
