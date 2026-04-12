package deployment

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
