package caddy

// ActiveDeployment is a denormalized view from a DB join for route building.
type ActiveDeployment struct {
	DeploymentID string
	ProjectID    string
	ProjectSlug  string
	Branch       string
	BranchSlug   string
	CommitSHA    string
	IsProduction bool
	ArtifactPath string
	Framework    string
}

// VerifiedDomain is a verified custom domain with its project context.
type VerifiedDomain struct {
	DomainID           string
	Domain             string
	ProjectID          string
	ProjectSlug        string
	ProductionArtifact string
	Framework          string
}
