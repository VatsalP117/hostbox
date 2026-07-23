package worker

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VatsalP117/hostbox/internal/models"
)

func TestGenerateDeploymentURL_Production(t *testing.T) {
	project := &models.Project{Slug: "my-app"}
	deployment := &models.Deployment{ID: "deploy12345678", IsProduction: true, CommitSHA: "abc123def456"}

	url := generateDeploymentURL(project, deployment, "example.com")
	expected := "https://my-app.example.com"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestCopyDirRejectsSymlink(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("host secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(src, "leak.txt")); err != nil {
		t.Fatal(err)
	}

	_, err := copyDir(src, dst)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symbolic-link rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "leak.txt")); !os.IsNotExist(err) {
		t.Fatalf("symlink target must not be copied, stat error = %v", err)
	}
}

func TestCopyDirRejectsNonRegularEntry(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	socketPath := filepath.Join(src, "build.sock")
	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	listener.SetUnlinkOnClose(false)
	defer listener.Close()

	_, err = copyDir(src, dst)
	if err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("expected non-regular entry rejection, got %v", err)
	}
}

func TestGenerateDeploymentURL_Preview(t *testing.T) {
	project := &models.Project{Slug: "my-app"}
	deployment := &models.Deployment{ID: "deploy12345678", IsProduction: false, CommitSHA: "abc123def456789"}

	url := generateDeploymentURL(project, deployment, "example.com")
	expected := "https://my-app-d-a27682.example.com"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestGenerateDeploymentURL_PreviewSanitizesDeploymentID(t *testing.T) {
	project := &models.Project{Slug: "app"}
	deployment := &models.Deployment{ID: "Deploy_ABC123456", IsProduction: false, CommitSHA: "ab12"}

	url := generateDeploymentURL(project, deployment, "host.io")
	expected := "https://app-d-c139e0.host.io"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
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

func TestCopyDirLimitedRejectsOversizedArtifact(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "large.txt"), []byte("123456"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := copyDirLimited(src, dst, 5); err == nil ||
		!strings.Contains(err.Error(), "maximum size") {
		t.Fatalf("expected artifact size rejection, got %v", err)
	}
}

func TestValidateArtifactTreeRejectsSymlinkAndSpecialFile(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		if _, err := validateArtifactTree(root, 1024); err == nil ||
			!strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("expected symlink rejection, got %v", err)
		}
	})

	t.Run("socket", func(t *testing.T) {
		root := t.TempDir()
		socketPath := filepath.Join(root, "artifact.sock")
		addr, err := net.ResolveUnixAddr("unix", socketPath)
		if err != nil {
			t.Fatal(err)
		}
		listener, err := net.ListenUnix("unix", addr)
		if err != nil {
			t.Skipf("unix sockets unavailable: %v", err)
		}
		listener.SetUnlinkOnClose(false)
		defer listener.Close()
		if _, err := validateArtifactTree(root, 1024); err == nil ||
			!strings.Contains(err.Error(), "non-regular") {
			t.Fatalf("expected special-file rejection, got %v", err)
		}
	})
}

func TestValidateArtifactTreeReturnsExactSize(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	size, err := validateArtifactTree(root, 5)
	if err != nil {
		t.Fatal(err)
	}
	if size != 5 {
		t.Fatalf("size = %d, want 5", size)
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
