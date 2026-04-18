package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type UpdateService struct {
	currentVersion string
	githubRepo     string
	httpClient     *http.Client
	logger         *slog.Logger
}

type UpdateCheck struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
	ReleaseNotes    string `json:"release_notes"`
	PublishedAt     string `json:"published_at"`
}

func NewUpdateService(currentVersion, githubRepo string, logger *slog.Logger) *UpdateService {
	if githubRepo == "" {
		githubRepo = "VatsalP117/hostbox"
	}
	return &UpdateService{
		currentVersion: currentVersion,
		githubRepo:     githubRepo,
		httpClient:     &http.Client{Timeout: 15 * time.Second},
		logger:         logger,
	}
}

func (s *UpdateService) Check(ctx context.Context) (*UpdateCheck, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", s.githubRepo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Hostbox/"+s.currentVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(s.currentVersion, "v")

	return &UpdateCheck{
		CurrentVersion:  s.currentVersion,
		LatestVersion:   release.TagName,
		UpdateAvailable: semverGreater(latest, current),
		ReleaseURL:      release.HTMLURL,
		ReleaseNotes:    release.Body,
		PublishedAt:     release.PublishedAt,
	}, nil
}

func (s *UpdateService) Execute(ctx context.Context, targetVersion string) error {
	s.logger.Info("starting update", "from", s.currentVersion, "to", targetVersion)

	s.logger.Info("pulling new images...")
	if err := s.runCommand(ctx, "docker", "compose", "pull"); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	s.logger.Info("updating containers...")
	if err := s.runCommand(ctx, "docker", "compose", "up", "-d", "--no-deps", "hostbox"); err != nil {
		return fmt.Errorf("failed to update containers: %w", err)
	}

	s.logger.Info("waiting for health check...")
	if err := s.waitForHealth(ctx, 60*time.Second); err != nil {
		s.logger.Error("health check failed, attempting rollback", "error", err)
		s.runCommand(ctx, "docker", "compose", "rollback") //nolint:errcheck
		return fmt.Errorf("update failed (rolled back): %w", err)
	}

	s.logger.Info("update completed successfully", "version", targetVersion)
	return nil
}

func (s *UpdateService) waitForHealth(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	healthURL := "http://localhost:8080/api/v1/health"

	for time.Now().Before(deadline) {
		resp, err := s.httpClient.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("health check timed out after %s", timeout)
}

func (s *UpdateService) runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = "/opt/hostbox"
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func semverGreater(a, b string) bool {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		va, _ := strconv.Atoi(safeIndex(partsA, i))
		vb, _ := strconv.Atoi(safeIndex(partsB, i))
		if va > vb {
			return true
		}
		if va < vb {
			return false
		}
	}
	return false
}

func safeIndex(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return "0"
}
