package github

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setupTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)

	_, pemBytes := generateTestKey(t)
	tp, err := NewTokenProviderWithBaseURL(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default(), server.URL)
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}

	// Pre-cache a token so we don't need the /app/installations/ endpoint
	tp.mu.Lock()
	tp.tokens[99] = &CachedToken{
		Token:     "ghs_test_token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	tp.mu.Unlock()

	client := NewClientWithBaseURL(tp, slog.Default(), server.URL)
	return client, server
}

func TestClient_ListInstallations(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app/installations" {
			json.NewEncoder(w).Encode([]Installation{
				{ID: 1, TargetType: "User"},
				{ID: 2, TargetType: "Organization"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tp, _ := NewTokenProviderWithBaseURL(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default(), server.URL)

	client := NewClientWithBaseURL(tp, slog.Default(), server.URL)
	installations, err := client.ListInstallations(context.Background())
	if err != nil {
		t.Fatalf("ListInstallations failed: %v", err)
	}
	if len(installations) != 2 {
		t.Errorf("expected 2 installations, got %d", len(installations))
	}
}

func TestClient_ListRepos(t *testing.T) {
	client, server := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/installation/repositories" {
			json.NewEncoder(w).Encode(listReposResponse{
				TotalCount: 2,
				Repositories: []Repository{
					{FullName: "user/repo1"},
					{FullName: "user/repo2"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	repos, total, err := client.ListRepos(context.Background(), 99, 1, 30)
	if err != nil {
		t.Fatalf("ListRepos failed: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(repos) != 2 {
		t.Errorf("repos = %d, want 2", len(repos))
	}
}

func TestClient_GetRepo(t *testing.T) {
	client, server := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/user/repo1" {
			json.NewEncoder(w).Encode(Repository{
				FullName:      "user/repo1",
				DefaultBranch: "main",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	repo, err := client.GetRepo(context.Background(), 99, "user", "repo1")
	if err != nil {
		t.Fatalf("GetRepo failed: %v", err)
	}
	if repo.FullName != "user/repo1" {
		t.Errorf("repo = %q, want user/repo1", repo.FullName)
	}
}

func TestClient_CreateDeployment(t *testing.T) {
	client, server := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(DeploymentResponse{ID: 42})
	})
	defer server.Close()

	resp, err := client.CreateDeployment(context.Background(), 99, "user", "repo1", CreateDeploymentRequest{
		Ref:         "main",
		Task:        "deploy",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("CreateDeployment failed: %v", err)
	}
	if resp.ID != 42 {
		t.Errorf("deployment ID = %d, want 42", resp.ID)
	}
}

func TestClient_RetryOn401(t *testing.T) {
	callCount := 0
	_, pemBytes := generateTestKey(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app/installations/99/access_tokens" {
			callCount++
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(installationTokenResponse{
				Token:     "ghs_refreshed",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			})
			return
		}
		if callCount < 1 {
			// First attempt returns 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// After token refresh, succeed
		json.NewEncoder(w).Encode(Repository{FullName: "user/repo1"})
	}))
	defer server.Close()

	tp, _ := NewTokenProviderWithBaseURL(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default(), server.URL)

	// Pre-cache an expired-ish token
	tp.mu.Lock()
	tp.tokens[99] = &CachedToken{Token: "old_token", ExpiresAt: time.Now().Add(1 * time.Minute)}
	tp.mu.Unlock()

	client := NewClientWithBaseURL(tp, slog.Default(), server.URL)
	repo, err := client.GetRepo(context.Background(), 99, "user", "repo1")
	if err != nil {
		t.Fatalf("GetRepo failed: %v", err)
	}
	if repo.FullName != "user/repo1" {
		t.Errorf("repo = %q", repo.FullName)
	}
}

func TestClient_PRComments(t *testing.T) {
	client, server := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			json.NewEncoder(w).Encode([]IssueComment{
				{ID: 1, Body: "existing comment"},
			})
		case "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(IssueComment{ID: 2, Body: "new comment"})
		case "PATCH":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	defer server.Close()

	ctx := context.Background()

	comments, err := client.ListPRComments(ctx, 99, "user", "repo1", 1)
	if err != nil {
		t.Fatalf("ListPRComments failed: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("comments = %d, want 1", len(comments))
	}

	comment, err := client.CreatePRComment(ctx, 99, "user", "repo1", 1, "new comment")
	if err != nil {
		t.Fatalf("CreatePRComment failed: %v", err)
	}
	if comment.ID != 2 {
		t.Errorf("comment ID = %d, want 2", comment.ID)
	}

	err = client.UpdateComment(ctx, 99, "user", "repo1", 1, "updated")
	if err != nil {
		t.Fatalf("UpdateComment failed: %v", err)
	}
}
