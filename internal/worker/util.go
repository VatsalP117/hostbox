package worker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vatsalpatel/hostbox/internal/models"
)

// generateDeploymentURL creates the URL for a deployment.
func generateDeploymentURL(project *models.Project, deployment *models.Deployment, platformDomain string) string {
	scheme := "https"

	if deployment.IsProduction {
		return fmt.Sprintf("%s://%s.%s", scheme, project.Slug, platformDomain)
	}

	shortSHA := deployment.CommitSHA
	if len(shortSHA) > 8 {
		shortSHA = shortSHA[:8]
	}

	return fmt.Sprintf("%s://%s-%s.%s", scheme, project.Slug, shortSHA, platformDomain)
}

// slugify converts a string to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

// copyDir recursively copies src to dst. Returns total bytes copied.
func copyDir(src, dst string) (int64, error) {
	var totalSize int64
	return totalSize, filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		totalSize += int64(len(data))
		return os.WriteFile(targetPath, data, 0644)
	})
}

// isDirEmpty checks if a directory has zero entries.
func isDirEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true, err
	}
	return len(entries) == 0, nil
}

// humanizeBytes formats bytes into a human-readable string.
func humanizeBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
