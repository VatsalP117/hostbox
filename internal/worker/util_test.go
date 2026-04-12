package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
)

func TestGenerateDeploymentURL_Production(t *testing.T) {
	project := &models.Project{Slug: "my-app"}
	deployment := &models.Deployment{IsProduction: true, CommitSHA: "abc123def456"}

	url := generateDeploymentURL(project, deployment, "example.com")
	expected := "https://my-app.example.com"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestGenerateDeploymentURL_Preview(t *testing.T) {
	project := &models.Project{Slug: "my-app"}
	deployment := &models.Deployment{IsProduction: false, CommitSHA: "abc123def456789"}

	url := generateDeploymentURL(project, deployment, "example.com")
	expected := "https://my-app-abc123de.example.com"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestGenerateDeploymentURL_ShortSHA(t *testing.T) {
	project := &models.Project{Slug: "app"}
	deployment := &models.Deployment{IsProduction: false, CommitSHA: "ab12"}

	url := generateDeploymentURL(project, deployment, "host.io")
	expected := "https://app-ab12.host.io"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"My--App", "my-app"},
		{"--leading-and-trailing--", "leading-and-trailing"},
		{"UPPER_CASE_123", "upper-case-123"},
		{"special!@#chars", "special-chars"},
		{"a", "a"},
		{"", ""},
		{"aaaaaaaaaa-bbbbbbbbbb-cccccccccc-dddddddddd-eeeeeeeeee", "aaaaaaaaaa-bbbbbbbbbb-cccccccccc-ddddddd"},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create source structure
	os.MkdirAll(filepath.Join(src, "subdir"), 0755)
	os.WriteFile(filepath.Join(src, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(src, "subdir", "file2.txt"), []byte("world!"), 0644)

	size, err := copyDir(src, dst)
	if err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	if size != 11 { // "hello" (5) + "world!" (6)
		t.Errorf("total size = %d, want 11", size)
	}

	data, err := os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil || string(data) != "hello" {
		t.Error("file1.txt not copied correctly")
	}

	data, err = os.ReadFile(filepath.Join(dst, "subdir", "file2.txt"))
	if err != nil || string(data) != "world!" {
		t.Error("subdir/file2.txt not copied correctly")
	}
}

func TestIsDirEmpty(t *testing.T) {
	empty := t.TempDir()
	isEmpty, err := isDirEmpty(empty)
	if err != nil || !isEmpty {
		t.Error("expected empty dir")
	}

	nonEmpty := t.TempDir()
	os.WriteFile(filepath.Join(nonEmpty, "file.txt"), []byte("x"), 0644)
	isEmpty, err = isDirEmpty(nonEmpty)
	if err != nil || isEmpty {
		t.Error("expected non-empty dir")
	}
}

func TestIsDirEmpty_NonExistent(t *testing.T) {
	isEmpty, err := isDirEmpty("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent dir")
	}
	if !isEmpty {
		t.Error("expected true for error case")
	}
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{2684354560, "2.5 GB"},
	}

	for _, tt := range tests {
		got := humanizeBytes(tt.input)
		if got != tt.expected {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
