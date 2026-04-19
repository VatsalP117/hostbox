package docker

import (
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
