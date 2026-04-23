package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

func TestGitHubHandler_ManifestExcludesUnsupportedDefaultEvents(t *testing.T) {
	db := setupTestDB(t)
	configStore := github.NewAppConfigStore(
		repository.NewSettingsRepository(db),
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)
	handler := NewGitHubHandler(
		github.NewRuntime(slog.Default()),
		configStore,
		"https://hostbox.example.com",
		"Hostbox",
		slog.Default(),
	)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/github/manifest", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Manifest(c); err != nil {
		t.Fatalf("Manifest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp githubManifestDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !strings.Contains(resp.ActionURL, "https://github.com/settings/apps/new?state=") {
		t.Fatalf("action_url = %q, want github app manifest URL", resp.ActionURL)
	}

	events, ok := resp.Manifest["default_events"].([]any)
	if !ok {
		t.Fatalf("default_events = %#v, want []any", resp.Manifest["default_events"])
	}

	var got []string
	for _, event := range events {
		name, ok := event.(string)
		if !ok {
			t.Fatalf("default_events item = %#v, want string", event)
		}
		got = append(got, name)
		if name == "installation" {
			t.Fatalf("default_events = %v, must not include unsupported installation event", got)
		}
	}

	want := []string{"push", "pull_request"}
	if len(got) != len(want) {
		t.Fatalf("default_events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("default_events = %v, want %v", got, want)
		}
	}
}
