package models

import "testing"

func TestDeploymentStatusStateMachine(t *testing.T) {
	statuses := []DeploymentStatus{
		DeploymentStatusQueued,
		DeploymentStatusBuilding,
		DeploymentStatusReady,
		DeploymentStatusFailed,
		DeploymentStatusCancelled,
	}
	allowed := map[[2]DeploymentStatus]bool{
		{DeploymentStatusQueued, DeploymentStatusBuilding}:    true,
		{DeploymentStatusQueued, DeploymentStatusFailed}:      true,
		{DeploymentStatusQueued, DeploymentStatusCancelled}:   true,
		{DeploymentStatusBuilding, DeploymentStatusReady}:     true,
		{DeploymentStatusBuilding, DeploymentStatusFailed}:    true,
		{DeploymentStatusBuilding, DeploymentStatusCancelled}: true,
		{DeploymentStatusReady, DeploymentStatusCancelled}:    true,
	}

	for _, from := range statuses {
		if !from.Valid() {
			t.Errorf("%q should be valid", from)
		}
		for _, to := range statuses {
			if got := from.CanTransitionTo(to); got != allowed[[2]DeploymentStatus{from, to}] {
				t.Errorf("%q -> %q = %v", from, to, got)
			}
		}
	}
	if DeploymentStatus("unknown").Valid() {
		t.Fatal("unknown status should be invalid")
	}
}
