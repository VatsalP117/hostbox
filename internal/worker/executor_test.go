package worker

import (
	"encoding/json"
	"testing"

	"github.com/VatsalP117/hostbox/internal/platform/detect"
)

func TestEffectiveBuildMemoryMB_BumpsWorkspaceDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkg := &detect.PackageJSON{Workspaces: json.RawMessage(`["apps/*"]`)}

	got := effectiveBuildMemoryMB(512, dir, pkg)
	if got != 1024 {
		t.Fatalf("expected workspace build memory to be bumped to 1024, got %d", got)
	}
}

func TestEffectiveBuildMemoryMB_PreservesConfiguredMemory(t *testing.T) {
	t.Parallel()

	got := effectiveBuildMemoryMB(1536, t.TempDir(), &detect.PackageJSON{})
	if got != 1536 {
		t.Fatalf("expected configured build memory to be preserved, got %d", got)
	}
}

func TestDescribeContainerExecError_AnnotatesOOMKill(t *testing.T) {
	t.Parallel()

	got := describeContainerExecError(assertErr("command exited with code 137"), 1024)
	want := "command exited with code 137 — build container was killed, likely due to memory pressure; increase BUILD_MEMORY_MB above 1024"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDescribeContainerExecError_PassesThroughOtherErrors(t *testing.T) {
	t.Parallel()

	got := describeContainerExecError(assertErr("command exited with code 1"), 1024)
	if got != "command exited with code 1" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func assertErr(msg string) error {
	return simpleError(msg)
}

type simpleError string

func (e simpleError) Error() string {
	return string(e)
}
