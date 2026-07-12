package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := New(server.URL, "test-token")
	return c
}

func requireRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method {
		t.Fatalf("method = %s, want %s", r.Method, method)
	}
	if r.URL.Path != path {
		t.Fatalf("path = %s, want %s", r.URL.Path, path)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("authorization = %q, want bearer token", got)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestDeploymentContracts(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodGet, "/api/v1/projects/project-1/deployments")
			writeJSON(t, w, map[string]any{"deployments": []map[string]any{{"id": "dep-1", "branch": "main"}}})
		})
		got, err := c.ListDeployments("project-1")
		if err != nil || len(got.Deployments) != 1 || got.Deployments[0].ID != "dep-1" {
			t.Fatalf("ListDeployments() = %#v, %v", got, err)
		}
	})

	t.Run("get uses global deployment route", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodGet, "/api/v1/deployments/dep-1")
			writeJSON(t, w, map[string]any{"deployment": map[string]any{"id": "dep-1", "status": "ready"}})
		})
		got, err := c.GetDeployment("dep-1")
		if err != nil || got.ID != "dep-1" || got.Status != "ready" {
			t.Fatalf("GetDeployment() = %#v, %v", got, err)
		}
	})

	t.Run("trigger uses build endpoint and branch body", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodPost, "/api/v1/projects/project-1/deployments/trigger")
			var body TriggerDeployRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body.Branch != "production" {
				t.Fatalf("branch = %q, want production", body.Branch)
			}
			w.WriteHeader(http.StatusAccepted)
			writeJSON(t, w, map[string]any{"deployment": map[string]any{"id": "dep-2", "branch": body.Branch}})
		})
		got, err := c.TriggerDeploy("project-1", TriggerDeployRequest{Branch: "production"})
		if err != nil || got.ID != "dep-2" || got.Branch != "production" {
			t.Fatalf("TriggerDeploy() = %#v, %v", got, err)
		}
	})

	t.Run("logs", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodGet, "/api/v1/deployments/dep-1/logs")
			writeJSON(t, w, map[string]any{"lines": []string{"install", "build"}, "total_lines": 3, "has_more": true})
		})
		got, err := c.GetDeploymentLogs("dep-1")
		if err != nil || len(got.Lines) != 2 || got.TotalLines != 3 || !got.HasMore {
			t.Fatalf("GetDeploymentLogs() = %#v, %v", got, err)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodPost, "/api/v1/projects/project-1/rollback")
			var body RollbackRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body.DeploymentID != "dep-old" {
				t.Fatalf("deployment_id = %q, want dep-old", body.DeploymentID)
			}
			writeJSON(t, w, map[string]any{"deployment": map[string]any{"id": "dep-new"}})
		})
		got, err := c.Rollback("project-1", RollbackRequest{DeploymentID: "dep-old"})
		if err != nil || got.ID != "dep-new" {
			t.Fatalf("Rollback() = %#v, %v", got, err)
		}
	})
}

func TestProjectAndAuthContracts(t *testing.T) {
	t.Run("login", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodPost, "/api/v1/auth/login")
			writeJSON(t, w, map[string]any{"access_token": "access", "user": map[string]any{"id": "user-1"}})
		})
		got, err := c.Login("user@example.com", "secret")
		if err != nil || got.AccessToken != "access" {
			t.Fatalf("Login() = %#v, %v", got, err)
		}
	})

	t.Run("whoami unwraps user", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodGet, "/api/v1/auth/me")
			writeJSON(t, w, map[string]any{"user": map[string]any{"id": "user-1", "email": "user@example.com", "is_admin": true}})
		})
		got, err := c.WhoAmI()
		if err != nil || got.ID != "user-1" || got.Email != "user@example.com" || !got.IsAdmin {
			t.Fatalf("WhoAmI() = %#v, %v", got, err)
		}
	})

	t.Run("list projects", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodGet, "/api/v1/projects")
			writeJSON(t, w, map[string]any{"projects": []map[string]any{{"id": "project-1", "production_branch": "main"}}})
		})
		got, err := c.ListProjects()
		if err != nil || len(got.Projects) != 1 || got.Projects[0].ProductionBranch != "main" {
			t.Fatalf("ListProjects() = %#v, %v", got, err)
		}
	})

	t.Run("get project unwraps detail response", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodGet, "/api/v1/projects/project-1")
			writeJSON(t, w, map[string]any{"project": map[string]any{"id": "project-1", "production_branch": "production", "github_repo": "acme/site"}})
		})
		got, err := c.GetProject("project-1")
		if err != nil || got.ProductionBranch != "production" || got.GitHubRepo == nil || *got.GitHubRepo != "acme/site" {
			t.Fatalf("GetProject() = %#v, %v", got, err)
		}
	})

	t.Run("create project uses backend field names", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodPost, "/api/v1/projects")
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body["github_repo"] != "acme/site" || body["output_directory"] != "dist" {
				t.Fatalf("unexpected project body: %#v", body)
			}
			if _, stale := body["git_repo"]; stale {
				t.Fatalf("stale git_repo field sent: %#v", body)
			}
			writeJSON(t, w, map[string]any{"project": map[string]any{"id": "project-1", "production_branch": "main"}})
		})
		got, err := c.CreateProject(CreateProjectRequest{Name: "Site", GitHubRepo: "acme/site", OutputDirectory: "dist"})
		if err != nil || got.ID != "project-1" {
			t.Fatalf("CreateProject() = %#v, %v", got, err)
		}
	})

	t.Run("update production branch", func(t *testing.T) {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			requireRequest(t, r, http.MethodPatch, "/api/v1/projects/project-1")
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body["production_branch"] != "production" {
				t.Fatalf("unexpected update body: %#v", body)
			}
			writeJSON(t, w, map[string]any{"project": map[string]any{"id": "project-1", "production_branch": "production"}})
		})
		got, err := c.UpdateProjectProductionBranch("project-1", "production")
		if err != nil || got.ProductionBranch != "production" {
			t.Fatalf("UpdateProjectProductionBranch() = %#v, %v", got, err)
		}
	})
}

func TestDomainEnvAndAdminContracts(t *testing.T) {
	t.Run("domains", func(t *testing.T) {
		requests := []struct{ method, path string }{
			{http.MethodGet, "/api/v1/projects/project-1/domains"},
			{http.MethodPost, "/api/v1/projects/project-1/domains"},
			{http.MethodDelete, "/api/v1/domains/domain-1"},
			{http.MethodPost, "/api/v1/domains/domain-1/verify"},
		}
		index := 0
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			want := requests[index]
			index++
			requireRequest(t, r, want.method, want.path)
			switch index {
			case 1:
				writeJSON(t, w, map[string]any{"domains": []any{}})
			case 2, 4:
				writeJSON(t, w, map[string]any{"domain": map[string]any{"id": "domain-1", "domain": "app.example.com", "verified": index == 4}})
			default:
				writeJSON(t, w, map[string]any{"success": true})
			}
		})
		if _, err := c.ListDomains("project-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddDomain("project-1", "app.example.com"); err != nil {
			t.Fatal(err)
		}
		if err := c.DeleteDomain("domain-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := c.VerifyDomain("domain-1"); err != nil {
			t.Fatal(err)
		}
		if index != len(requests) {
			t.Fatalf("requests = %d, want %d", index, len(requests))
		}
	})

	t.Run("environment variables", func(t *testing.T) {
		requests := []struct{ method, path string }{
			{http.MethodGet, "/api/v1/projects/project-1/env-vars"},
			{http.MethodPost, "/api/v1/projects/project-1/env-vars"},
			{http.MethodDelete, "/api/v1/env-vars/env-1"},
		}
		index := 0
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			want := requests[index]
			index++
			requireRequest(t, r, want.method, want.path)
			if index == 1 {
				writeJSON(t, w, map[string]any{"env_vars": []map[string]any{{"id": "env-1", "key": "API_URL"}}})
			} else {
				writeJSON(t, w, map[string]any{"success": true})
			}
		})
		got, err := c.ListEnvVars("project-1")
		if err != nil || len(got.EnvVars) != 1 {
			t.Fatalf("ListEnvVars() = %#v, %v", got, err)
		}
		if err := c.SetEnvVar("project-1", "API_URL", "https://example.com"); err != nil {
			t.Fatal(err)
		}
		if err := c.DeleteEnvVar("env-1"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("admin backups", func(t *testing.T) {
		requests := []struct{ method, path string }{
			{http.MethodPost, "/api/v1/admin/backups"},
			{http.MethodGet, "/api/v1/admin/backups"},
			{http.MethodPost, "/api/v1/admin/backups/restore"},
		}
		index := 0
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			want := requests[index]
			index++
			requireRequest(t, r, want.method, want.path)
			if index == 1 {
				if r.URL.Query().Get("compress") != "false" {
					t.Fatalf("compress query = %q", r.URL.Query().Get("compress"))
				}
				writeJSON(t, w, map[string]any{"backup": map[string]any{"filename": "hostbox.db", "size_bytes": 42}})
			} else if index == 2 {
				writeJSON(t, w, map[string]any{"backups": []map[string]any{{"filename": "hostbox.db"}}})
			} else {
				writeJSON(t, w, map[string]any{"message": "restored"})
			}
		})
		backup, err := c.CreateBackup(false)
		if err != nil || backup.Filename != "hostbox.db" || backup.SizeBytes != 42 {
			t.Fatalf("CreateBackup() = %#v, %v", backup, err)
		}
		backups, err := c.ListBackups()
		if err != nil || len(backups.Backups) != 1 {
			t.Fatalf("ListBackups() = %#v, %v", backups, err)
		}
		if err := c.RestoreBackup("/backups/hostbox.db"); err != nil {
			t.Fatal(err)
		}
	})
}
