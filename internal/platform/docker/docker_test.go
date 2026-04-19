package docker

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/mount"
)

func TestBuildContainerMounts(t *testing.T) {
	t.Parallel()

	mounts := buildContainerMounts(BuildContainerOpts{
		SourceDir:   "/tmp/source",
		OutputDir:   "/tmp/output",
		CacheVolume: "modules-cache",
		BuildCache:  "build-cache",
	})

	if len(mounts) != 4 {
		t.Fatalf("expected 4 mounts, got %d", len(mounts))
	}

	sourceMount := findMount(t, mounts, buildSourceDir)
	if sourceMount.Type != mount.TypeBind {
		t.Fatalf("expected source mount to be a bind mount, got %s", sourceMount.Type)
	}
	if sourceMount.ReadOnly {
		t.Fatalf("expected source mount to be writable")
	}

	nodeModulesMount := findMount(t, mounts, buildNodeModulesDir)
	if nodeModulesMount.Type != mount.TypeVolume {
		t.Fatalf("expected node_modules mount to be a volume, got %s", nodeModulesMount.Type)
	}
	if nodeModulesMount.Source != "modules-cache" {
		t.Fatalf("expected node_modules volume source modules-cache, got %s", nodeModulesMount.Source)
	}

	buildCacheMount := findMount(t, mounts, buildCacheDir)
	if buildCacheMount.Type != mount.TypeVolume {
		t.Fatalf("expected build cache mount to be a volume, got %s", buildCacheMount.Type)
	}
	if buildCacheMount.Source != "build-cache" {
		t.Fatalf("expected build cache volume source build-cache, got %s", buildCacheMount.Source)
	}
}

func TestWrapBuildCommandSetsBuildToolShims(t *testing.T) {
	t.Parallel()

	wrapped := wrapBuildCommand("pnpm install --frozen-lockfile && bun run build")

	required := []string{
		`export HOME=/app/.build-cache/home`,
		`export XDG_CACHE_HOME=/app/.build-cache/xdg-cache`,
		`export COREPACK_HOME=/app/.build-cache/corepack`,
		`export NPM_CONFIG_CACHE=/app/.build-cache/npm`,
		`export YARN_CACHE_FOLDER=/app/.build-cache/yarn`,
		`export PATH="/app/.build-cache/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"`,
		`printf '%s\n' '#!/bin/sh' 'exec corepack pnpm "$@"' > /app/.build-cache/bin/pnpm`,
		`printf '%s\n' '#!/bin/sh' 'exec corepack yarn "$@"' > /app/.build-cache/bin/yarn`,
		`printf '%s\n' '#!/bin/sh' 'exec npx --yes bun@1 "$@"' > /app/.build-cache/bin/bun`,
		`chmod +x /app/.build-cache/bin/pnpm /app/.build-cache/bin/yarn /app/.build-cache/bin/bun`,
		`pnpm install --frozen-lockfile && bun run build`,
	}

	for _, want := range required {
		if !strings.Contains(wrapped, want) {
			t.Fatalf("wrapped command missing %q:\n%s", want, wrapped)
		}
	}
}

func TestExtractTarStripsCopiedRootDirectory(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	writeTarDir(t, tw, "out")
	writeTarFile(t, tw, "out/index.html", "<html>ok</html>")
	writeTarDir(t, tw, "out/assets")
	writeTarFile(t, tw, "out/assets/app.js", "console.log('ok');")

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	destDir := t.TempDir()
	size, err := extractTar(bytes.NewReader(buf.Bytes()), destDir, "out")
	if err != nil {
		t.Fatalf("extract tar: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "index.html")); err != nil {
		t.Fatalf("expected index.html at artifact root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "assets", "app.js")); err != nil {
		t.Fatalf("expected nested asset at artifact root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "out")); !os.IsNotExist(err) {
		t.Fatalf("expected copied root directory to be stripped, got err=%v", err)
	}

	wantSize := int64(len("<html>ok</html>") + len("console.log('ok');"))
	if size != wantSize {
		t.Fatalf("extracted size = %d, want %d", size, wantSize)
	}
}

func TestExtractTarRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	writeTarFile(t, tw, "../escape.txt", "bad")
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	if _, err := extractTar(bytes.NewReader(buf.Bytes()), t.TempDir(), ""); err == nil {
		t.Fatal("expected traversal error")
	}
}

func findMount(t *testing.T, mounts []mount.Mount, target string) mount.Mount {
	t.Helper()

	for _, m := range mounts {
		if m.Target == target {
			return m
		}
	}

	t.Fatalf("mount with target %s not found", target)
	return mount.Mount{}
}

func writeTarDir(t *testing.T, tw *tar.Writer, name string) {
	t.Helper()

	if err := tw.WriteHeader(&tar.Header{
		Name:     name,
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}); err != nil {
		t.Fatalf("write tar dir %s: %v", name, err)
	}
}

func writeTarFile(t *testing.T, tw *tar.Writer, name, contents string) {
	t.Helper()

	if err := tw.WriteHeader(&tar.Header{
		Name:     name,
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(contents)),
	}); err != nil {
		t.Fatalf("write tar file header %s: %v", name, err)
	}
	if _, err := tw.Write([]byte(contents)); err != nil {
		t.Fatalf("write tar file body %s: %v", name, err)
	}
}
