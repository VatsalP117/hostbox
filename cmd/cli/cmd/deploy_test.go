package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	clientpkg "github.com/VatsalP117/hostbox/cmd/cli/internal/client"
)

func TestResolveDeployBranch(t *testing.T) {
	t.Run("uses explicit branch without a request", func(t *testing.T) {
		got, err := resolveDeployBranch(nil, "project-1", "preview")
		if err != nil || got != "preview" {
			t.Fatalf("resolveDeployBranch() = %q, %v", got, err)
		}
	})

	t.Run("loads production branch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/projects/project-1" {
				t.Fatalf("request = %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"project":{"id":"project-1","production_branch":"production"}}`))
		}))
		defer server.Close()

		got, err := resolveDeployBranch(clientpkg.New(server.URL, "token"), "project-1", "")
		if err != nil || got != "production" {
			t.Fatalf("resolveDeployBranch() = %q, %v", got, err)
		}
	})

	t.Run("rejects empty configured branch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"project":{"id":"project-1","production_branch":""}}`))
		}))
		defer server.Close()

		_, err := resolveDeployBranch(clientpkg.New(server.URL, "token"), "project-1", "")
		if err == nil {
			t.Fatal("resolveDeployBranch() error = nil")
		}
	})
}
