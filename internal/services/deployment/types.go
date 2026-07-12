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
}

// ListOpts configures list queries.
type ListOpts struct {
	Page    int
	PerPage int
	Status  *string // filter by status (nil = all)
	Branch  *string // filter by branch (nil = all)
}
