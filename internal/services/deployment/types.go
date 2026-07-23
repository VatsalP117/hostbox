package deployment

import "context"

// ProductionActivation identifies the artifact that should serve a project's
// production hostname.
type ProductionActivation struct {
	ProjectID    string
	ProjectSlug  string
	ArtifactPath string
	Framework    string
}

// ProductionActivator applies a production route change before rollback or
// promotion is reported as successful.
type ProductionActivator interface {
	ActivateProduction(ctx context.Context, activation ProductionActivation) error
}

// TriggerRequest holds the data needed to trigger a deployment.
type TriggerRequest struct {
	ProjectID     string
	Branch        string
	CommitSHA     string
	CommitMessage *string
	CommitAuthor  *string
	PRNumber      *int
	BuildManifest *BuildManifest
	IsProduction  *bool
}

// BuildManifest is the immutable build recipe attached to a deployment.
// Unresolved manifests contain the project settings captured at queue time;
// resolved manifests contain the exact effective recipe used by the worker.
type BuildManifest struct {
	Framework             *string
	ServingMode           *string
	PackageManager        *string
	PackageManagerVersion *string
	NodeVersion           string
	RootDirectory         string
	OutputDirectory       *string
	InstallCommand        *string
	BuildCommand          *string
	LockFileHash          string
	Resolved              bool
	SourceRepository      *string
	SourceInstallationID  *int64
}

// ListOpts configures list queries.
type ListOpts struct {
	Page    int
	PerPage int
	Status  *string // filter by status (nil = all)
	Branch  *string // filter by branch (nil = all)
}
