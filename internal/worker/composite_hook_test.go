package worker

import (
	"context"
	"testing"

	"github.com/VatsalP117/hostbox/internal/models"
)

func TestCompositePostBuildHookForwardsCancellationOnlyToCleanupHooks(t *testing.T) {
	cleanup := &recordingCancellationHook{}
	ordinary := &recordingPostBuildHook{}
	composite := NewCompositePostBuildHook(ordinary, cleanup)

	if err := composite.OnBuildCancelled(context.Background(), &models.Project{}, &models.Deployment{}); err != nil {
		t.Fatalf("OnBuildCancelled: %v", err)
	}
	if cleanup.cancelled != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleanup.cancelled)
	}
}
