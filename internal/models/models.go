package models

import "time"

// --- User ---

type User struct {
	ID                              string     `db:"id"`
	Email                           string     `db:"email"`
	PasswordHash                    string     `db:"password_hash"`
	DisplayName                     *string    `db:"display_name"`
	IsAdmin                         bool       `db:"is_admin"`
	EmailVerified                   bool       `db:"email_verified"`
	ResetTokenHash                  *string    `db:"reset_token_hash"`
	ResetTokenExpiresAt             *time.Time `db:"reset_token_expires_at"`
	EmailVerificationTokenHash      *string    `db:"email_verification_token_hash"`
	EmailVerificationTokenExpiresAt *time.Time `db:"email_verification_token_expires_at"`
	CreatedAt                       time.Time  `db:"created_at"`
	UpdatedAt                       time.Time  `db:"updated_at"`
}

// --- Session ---

type Session struct {
	ID               string    `db:"id"`
	UserID           string    `db:"user_id"`
	RefreshTokenHash string    `db:"refresh_token_hash"`
	UserAgent        *string   `db:"user_agent"`
	IPAddress        *string   `db:"ip_address"`
	ExpiresAt        time.Time `db:"expires_at"`
	CreatedAt        time.Time `db:"created_at"`
}

// --- Project ---

type Project struct {
	ID                     string    `db:"id"`
	OwnerID                string    `db:"owner_id"`
	Name                   string    `db:"name"`
	Slug                   string    `db:"slug"`
	GitHubRepo             *string   `db:"github_repo"`
	GitHubInstallationID   *int64    `db:"github_installation_id"`
	ProductionBranch       string    `db:"production_branch"`
	Framework              *string   `db:"framework"`
	BuildCommand           *string   `db:"build_command"`
	InstallCommand         *string   `db:"install_command"`
	OutputDirectory        *string   `db:"output_directory"`
	RootDirectory          string    `db:"root_directory"`
	NodeVersion            string    `db:"node_version"`
	AutoDeploy             bool      `db:"auto_deploy"`
	PreviewDeployments     bool      `db:"preview_deployments"`
	LockFileHash           string    `db:"lock_file_hash"`
	DetectedPackageManager string    `db:"detected_package_manager"`
	CreatedAt              time.Time `db:"created_at"`
	UpdatedAt              time.Time `db:"updated_at"`
}

// --- Deployment ---

// DeploymentStatus represents the state of a deployment.
type DeploymentStatus string

const (
	DeploymentStatusQueued    DeploymentStatus = "queued"
	DeploymentStatusBuilding  DeploymentStatus = "building"
	DeploymentStatusReady     DeploymentStatus = "ready"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusCancelled DeploymentStatus = "cancelled"
)

type Deployment struct {
	ID                string           `db:"id"`
	ProjectID         string           `db:"project_id"`
	CommitSHA         string           `db:"commit_sha"`
	CommitMessage     *string          `db:"commit_message"`
	CommitAuthor      *string          `db:"commit_author"`
	Branch            string           `db:"branch"`
	Status            DeploymentStatus `db:"status"`
	IsProduction      bool             `db:"is_production"`
	DeploymentURL     *string          `db:"deployment_url"`
	ArtifactPath      *string          `db:"artifact_path"`
	ArtifactSizeBytes *int64           `db:"artifact_size_bytes"`
	LogPath           *string          `db:"log_path"`
	ErrorMessage      *string          `db:"error_message"`
	IsRollback        bool             `db:"is_rollback"`
	RollbackSourceID  *string          `db:"rollback_source_id"`
	GitHubPRNumber    *int             `db:"github_pr_number"`
	GitHubDeployID    *int64           `db:"github_deploy_id"`
	BuildDurationMs   *int64           `db:"build_duration_ms"`
	StartedAt         *time.Time       `db:"started_at"`
	CompletedAt       *time.Time       `db:"completed_at"`
	CreatedAt         time.Time        `db:"created_at"`
}

// --- Domain ---

type Domain struct {
	ID            string     `db:"id"`
	ProjectID     string     `db:"project_id"`
	Domain        string     `db:"domain"`
	Verified      bool       `db:"verified"`
	VerifiedAt    *time.Time `db:"verified_at"`
	LastCheckedAt *time.Time `db:"last_checked_at"`
	CreatedAt     time.Time  `db:"created_at"`
}

// --- EnvVar ---

type EnvVar struct {
	ID             string    `db:"id"`
	ProjectID      string    `db:"project_id"`
	Key            string    `db:"key"`
	EncryptedValue string    `db:"encrypted_value"`
	IsSecret       bool      `db:"is_secret"`
	Scope          string    `db:"scope"` // "all", "preview", "production"
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// --- NotificationConfig ---

type NotificationConfig struct {
	ID         string    `db:"id"`
	ProjectID  *string   `db:"project_id"`
	Channel    string    `db:"channel"` // "discord", "slack", "webhook"
	WebhookURL string    `db:"webhook_url"`
	Events     string    `db:"events"` // comma-separated or "all"
	Enabled    bool      `db:"enabled"`
	CreatedAt  time.Time `db:"created_at"`
}

// --- ActivityLog ---

type ActivityLog struct {
	ID           int64     `db:"id"`
	UserID       *string   `db:"user_id"`
	Action       string    `db:"action"`        // e.g., "deployment.created"
	ResourceType string    `db:"resource_type"` // e.g., "project"
	ResourceID   *string   `db:"resource_id"`
	Metadata     *string   `db:"metadata"` // JSON string
	CreatedAt    time.Time `db:"created_at"`
}

// --- Setting ---

type Setting struct {
	Key       string    `db:"key"`
	Value     string    `db:"value"`
	UpdatedAt time.Time `db:"updated_at"`
}

// --- SystemMetricSnapshot ---

type SystemMetricSnapshot struct {
	ID                   int64     `db:"id"`
	CPUUsagePercent      float64   `db:"cpu_usage_percent"`
	Load1                float64   `db:"load1"`
	Load5                float64   `db:"load5"`
	Load15               float64   `db:"load15"`
	MemoryUsedBytes      int64     `db:"memory_used_bytes"`
	MemoryTotalBytes     int64     `db:"memory_total_bytes"`
	MemoryAvailableBytes int64     `db:"memory_available_bytes"`
	MemoryUsagePercent   float64   `db:"memory_usage_percent"`
	DiskUsedBytes        int64     `db:"disk_used_bytes"`
	DiskTotalBytes       int64     `db:"disk_total_bytes"`
	DiskAvailableBytes   int64     `db:"disk_available_bytes"`
	DiskUsagePercent     float64   `db:"disk_usage_percent"`
	ActiveBuilds         int64     `db:"active_builds"`
	QueuedBuilds         int64     `db:"queued_builds"`
	CreatedAt            time.Time `db:"created_at"`
}
