package config

import (
	"os"
	"testing"
)

func setTestRoot(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	configRoot = dir
	t.Cleanup(func() { configRoot = "" })
}

func TestLoadMissing(t *testing.T) {
	setTestRoot(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerURL != "" {
		t.Errorf("expected empty ServerURL, got %q", cfg.ServerURL)
	}
}

func TestSaveAndLoad(t *testing.T) {
	setTestRoot(t)
	cfg := &Config{ServerURL: "https://test.example.com"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ServerURL != "https://test.example.com" {
		t.Errorf("expected https://test.example.com, got %q", loaded.ServerURL)
	}
}

func TestTokenSaveLoadClear(t *testing.T) {
	setTestRoot(t)

	// Initially empty
	tok, err := LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if tok != "" {
		t.Errorf("expected empty token, got %q", tok)
	}

	// Save
	if err := SaveToken("my-secret-token"); err != nil {
		t.Fatal(err)
	}

	// Load
	tok, err = LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if tok != "my-secret-token" {
		t.Errorf("expected my-secret-token, got %q", tok)
	}

	// Check permissions
	info, err := os.Stat(tokenFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 perms, got %o", info.Mode().Perm())
	}

	// Clear
	if err := ClearToken(); err != nil {
		t.Fatal(err)
	}
	tok, err = LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if tok != "" {
		t.Errorf("expected empty token after clear, got %q", tok)
	}
}

func TestClearTokenNonExistent(t *testing.T) {
	setTestRoot(t)
	if err := ClearToken(); err != nil {
		t.Errorf("ClearToken on non-existent should not error: %v", err)
	}
}
