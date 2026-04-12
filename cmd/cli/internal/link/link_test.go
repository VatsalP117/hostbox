package link

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndDiscover(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	os.Chdir(dir)

	if err := Save("proj-123", "https://hb.example.com"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectID != "proj-123" {
		t.Errorf("expected proj-123, got %q", cfg.ProjectID)
	}
	if cfg.ServerURL != "https://hb.example.com" {
		t.Errorf("expected https://hb.example.com, got %q", cfg.ServerURL)
	}
}

func TestDiscoverWalksUp(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	// Write link in parent
	os.Chdir(dir)
	if err := Save("proj-parent", ""); err != nil {
		t.Fatal(err)
	}

	// Create and chdir to child
	child := filepath.Join(dir, "subdir", "deep")
	os.MkdirAll(child, 0755)
	os.Chdir(child)

	cfg, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectID != "proj-parent" {
		t.Errorf("expected proj-parent, got %q", cfg.ProjectID)
	}
}

func TestDiscoverNoLink(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	os.Chdir(dir)

	_, err := Discover()
	if err == nil {
		t.Fatal("expected error when no .hostbox.json exists")
	}
}
