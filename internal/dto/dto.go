package dto

// --- Pagination ---

type PaginationQuery struct {
	Page    int `query:"page" validate:"omitempty,min=1"`
	PerPage int `query:"per_page" validate:"omitempty,min=1,max=100"`
}

func (p PaginationQuery) PageOrDefault() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

func (p PaginationQuery) PerPageOrDefault() int {
	if p.PerPage < 1 || p.PerPage > 100 {
		return 20
	}
	return p.PerPage
}

func (p PaginationQuery) Offset() int {
	return (p.PageOrDefault() - 1) * p.PerPageOrDefault()
}

type PaginationResponse struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

func NewPaginationResponse(total, page, perPage int) PaginationResponse {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	return PaginationResponse{
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}

// --- Auth ---

type RegisterRequest struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	Password    string  `json:"password" validate:"required,min=8,max=128"`
	DisplayName *string `json:"display_name" validate:"omitempty,min=1,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=1,max=100"`
	Email       *string `json:"email" validate:"omitempty,email,max=255"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// --- Projects ---

type CreateProjectRequest struct {
	Name                 string  `json:"name" validate:"required,min=1,max=100"`
	GitHubRepo           *string `json:"github_repo" validate:"omitempty,max=200"`
	GitHubInstallationID *int64  `json:"github_installation_id"`
	BuildCommand         *string `json:"build_command" validate:"omitempty,max=500"`
	InstallCommand       *string `json:"install_command" validate:"omitempty,max=500"`
	OutputDirectory      *string `json:"output_directory" validate:"omitempty,max=200"`
	RootDirectory        *string `json:"root_directory" validate:"omitempty,max=200"`
	NodeVersion          *string `json:"node_version" validate:"omitempty,oneof=18 20 22"`
}

type UpdateProjectRequest struct {
	Name               *string `json:"name" validate:"omitempty,min=1,max=100"`
	BuildCommand       *string `json:"build_command" validate:"omitempty,max=500"`
	InstallCommand     *string `json:"install_command" validate:"omitempty,max=500"`
	OutputDirectory    *string `json:"output_directory" validate:"omitempty,max=200"`
	RootDirectory      *string `json:"root_directory" validate:"omitempty,max=200"`
	NodeVersion        *string `json:"node_version" validate:"omitempty,oneof=18 20 22"`
	ProductionBranch   *string `json:"production_branch" validate:"omitempty,min=1,max=200"`
	AutoDeploy         *bool   `json:"auto_deploy"`
	PreviewDeployments *bool   `json:"preview_deployments"`
}

// --- Deployments ---

type CreateDeploymentRequest struct {
	Branch    *string `json:"branch" validate:"omitempty,min=1,max=200"`
	CommitSHA *string `json:"commit_sha" validate:"omitempty,len=40"`
}

type TriggerDeployRequest struct {
	Branch        string  `json:"branch" validate:"required,min=1,max=200"`
	CommitSHA     string  `json:"commit_sha" validate:"omitempty,max=40"`
	CommitMessage *string `json:"commit_message,omitempty" validate:"omitempty,max=500"`
	CommitAuthor  *string `json:"commit_author,omitempty" validate:"omitempty,max=100"`
}

type RollbackRequest struct {
	DeploymentID string `json:"deployment_id" validate:"required"`
}

type ListDeploymentsQuery struct {
	PaginationQuery
	Status *string `query:"status" validate:"omitempty,oneof=queued building ready failed cancelled"`
	Branch *string `query:"branch" validate:"omitempty,min=1"`
}

// --- Domains ---

type CreateDomainRequest struct {
	Domain string `json:"domain" validate:"required,fqdn,max=253"`
}

// --- Env Vars ---

type CreateEnvVarRequest struct {
	Key      string  `json:"key" validate:"required,min=1,max=255"`
	Value    string  `json:"value" validate:"required"`
	IsSecret *bool   `json:"is_secret"`
	Scope    *string `json:"scope" validate:"omitempty,oneof=all preview production"`
}

type BulkEnvVarRequest struct {
	EnvVars []CreateEnvVarRequest `json:"env_vars" validate:"required,min=1,dive"`
}

type UpdateEnvVarRequest struct {
	Value    *string `json:"value"`
	IsSecret *bool   `json:"is_secret"`
	Scope    *string `json:"scope" validate:"omitempty,oneof=all preview production"`
}

// --- Notifications ---

type CreateNotificationRequest struct {
	Channel    string  `json:"channel" validate:"required,oneof=discord slack webhook"`
	WebhookURL string  `json:"webhook_url" validate:"required,url,max=500"`
	Events     *string `json:"events" validate:"omitempty,max=500"`
}

type UpdateNotificationRequest struct {
	WebhookURL *string `json:"webhook_url" validate:"omitempty,url,max=500"`
	Events     *string `json:"events" validate:"omitempty,max=500"`
	Enabled    *bool   `json:"enabled"`
}

// --- Admin ---

type UpdateSettingsRequest struct {
	RegistrationEnabled   *bool `json:"registration_enabled"`
	MaxProjects           *int  `json:"max_projects" validate:"omitempty,min=1,max=10000"`
	MaxConcurrentBuilds   *int  `json:"max_concurrent_builds" validate:"omitempty,min=1,max=10"`
	ArtifactRetentionDays *int  `json:"artifact_retention_days" validate:"omitempty,min=1,max=365"`
}

// --- Response DTOs ---

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type UserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	DisplayName   *string `json:"display_name,omitempty"`
	IsAdmin       bool    `json:"is_admin"`
	EmailVerified bool    `json:"email_verified"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type AuthResponse struct {
	User        UserResponse `json:"user"`
	AccessToken string       `json:"access_token"`
}

type ProjectResponse struct {
	ID                   string  `json:"id"`
	OwnerID              string  `json:"owner_id"`
	Name                 string  `json:"name"`
	Slug                 string  `json:"slug"`
	GitHubRepo           *string `json:"github_repo,omitempty"`
	GitHubInstallationID *int64  `json:"github_installation_id,omitempty"`
	ProductionBranch     string  `json:"production_branch"`
	Framework            *string `json:"framework,omitempty"`
	BuildCommand         *string `json:"build_command,omitempty"`
	InstallCommand       *string `json:"install_command,omitempty"`
	OutputDirectory      *string `json:"output_directory,omitempty"`
	RootDirectory        string  `json:"root_directory"`
	NodeVersion          string  `json:"node_version"`
	AutoDeploy           bool    `json:"auto_deploy"`
	PreviewDeployments   bool    `json:"preview_deployments"`
	Status               string  `json:"status"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

type ProjectListResponse struct {
	Projects   []ProjectResponse  `json:"projects"`
	Pagination PaginationResponse `json:"pagination"`
}

type DeploymentResponse struct {
	ID                string  `json:"id"`
	ProjectID         string  `json:"project_id"`
	CommitSHA         string  `json:"commit_sha"`
	CommitMessage     *string `json:"commit_message,omitempty"`
	CommitAuthor      *string `json:"commit_author,omitempty"`
	Branch            string  `json:"branch"`
	Status            string  `json:"status"`
	IsProduction      bool    `json:"is_production"`
	DeploymentURL     *string `json:"deployment_url,omitempty"`
	ArtifactSizeBytes *int64  `json:"artifact_size_bytes,omitempty"`
	ErrorMessage      *string `json:"error_message,omitempty"`
	IsRollback        bool    `json:"is_rollback"`
	RollbackSourceID  *string `json:"rollback_source_id,omitempty"`
	GitHubPRNumber    *int    `json:"github_pr_number,omitempty"`
	BuildDurationMs   *int64  `json:"build_duration_ms,omitempty"`
	StartedAt         *string `json:"started_at,omitempty"`
	CompletedAt       *string `json:"completed_at,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type DeploymentListResponse struct {
	Deployments []DeploymentResponse `json:"deployments"`
	Pagination  PaginationResponse   `json:"pagination"`
}

type LogResponse struct {
	Lines      []string `json:"lines"`
	TotalLines int      `json:"total_lines"`
	HasMore    bool     `json:"has_more"`
}

type DomainResponse struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	Domain        string  `json:"domain"`
	Verified      bool    `json:"verified"`
	VerifiedAt    *string `json:"verified_at,omitempty"`
	LastCheckedAt *string `json:"last_checked_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type EnvVarResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"` // "••••••••" for secrets
	IsSecret  bool   `json:"is_secret"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type NotificationConfigResponse struct {
	ID         string  `json:"id"`
	ProjectID  *string `json:"project_id,omitempty"`
	Channel    string  `json:"channel"`
	WebhookURL string  `json:"webhook_url"`
	Events     string  `json:"events"`
	Enabled    bool    `json:"enabled"`
	CreatedAt  string  `json:"created_at"`
}

type ProjectStatsResponse struct {
	TotalDeployments   int64  `json:"total_deployments"`
	SuccessfulBuilds   int64  `json:"successful_builds"`
	FailedBuilds       int64  `json:"failed_builds"`
	AverageBuildTimeMs *int64 `json:"average_build_time_ms,omitempty"`
	LastDeployAt       *string `json:"last_deploy_at,omitempty"`
}

type ProjectDetailResponse struct {
	Project            ProjectResponse      `json:"project"`
	LatestDeployment   *DeploymentResponse  `json:"latest_deployment"`
	Domains            []DomainResponse     `json:"domains"`
	Stats              ProjectStatsResponse `json:"stats"`
	EnvVarsCount       int64                `json:"env_vars_count"`
	NotificationsCount int64                `json:"notifications_count"`
}

type ActivityLogResponse struct {
	ID           int64   `json:"id"`
	UserID       *string `json:"user_id,omitempty"`
	UserName     *string `json:"user_name,omitempty"`
	Action       string  `json:"action"`
	ResourceType string  `json:"resource_type"`
	ResourceID   *string `json:"resource_id,omitempty"`
	ProjectName  *string `json:"project_name,omitempty"`
	Metadata     *string `json:"metadata,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type ActivityListResponse struct {
	Activities []ActivityLogResponse `json:"activities"`
	Pagination PaginationResponse    `json:"pagination"`
}

type HealthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

type SettingsResponse struct {
	SetupComplete            string `json:"setup_complete"`
	RegistrationEnabled      string `json:"registration_enabled"`
	MaxProjects              string `json:"max_projects"`
	MaxDeploymentsPerProject string `json:"max_deployments_per_project"`
	ArtifactRetentionDays    string `json:"artifact_retention_days"`
	MaxConcurrentBuilds      string `json:"max_concurrent_builds"`
}

// --- Additional Auth DTOs ---

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type LogoutAllResponse struct {
	Success         bool `json:"success"`
	SessionsRevoked int  `json:"sessions_revoked"`
}

// --- Setup DTOs ---

type SetupRequest struct {
	Email       string `json:"email" validate:"required,email,max=255"`
	Password    string `json:"password" validate:"required,min=8,max=128"`
	DisplayName string `json:"display_name" validate:"omitempty,max=100"`
}

type SetupStatusResponse struct {
	SetupRequired bool `json:"setup_required"`
}

// --- Admin DTOs ---

type AdminStatsResponse struct {
	ProjectCount        int64                    `json:"project_count"`
	DeploymentCount     int64                    `json:"deployment_count"`
	ActiveBuilds        int64                    `json:"active_builds"`
	QueuedBuilds        int64                    `json:"queued_builds"`
	MaxConcurrentBuilds int64                    `json:"max_concurrent_builds"`
	UserCount           int64                    `json:"user_count"`
	DiskUsage           DiskUsageResponse        `json:"disk_usage"`
	Components          ComponentHealthResponse  `json:"components"`
	CPU                 CPUStatsResponse         `json:"cpu"`
	Memory              MemoryStatsResponse      `json:"memory"`
	BuildQueue          BuildQueueResponse       `json:"build_queue"`
	DeploymentHealth    DeploymentHealthResponse `json:"deployment_health"`
	Trends              MetricTrendsResponse     `json:"trends"`
	Alerts              []SystemAlertResponse    `json:"alerts"`
	UptimeSeconds       int64                    `json:"uptime_seconds"`
}

type ComponentHealthResponse struct {
	API      ServiceHealthResponse `json:"api"`
	Database ServiceHealthResponse `json:"database"`
	Docker   ServiceHealthResponse `json:"docker"`
	Caddy    ServiceHealthResponse `json:"caddy"`
}

type ServiceHealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type CPUStatsResponse struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
	Load1        float64 `json:"load1"`
	Load5        float64 `json:"load5"`
	Load15       float64 `json:"load15"`
}

type MemoryStatsResponse struct {
	UsedBytes      int64   `json:"used_bytes"`
	TotalBytes     int64   `json:"total_bytes"`
	AvailableBytes int64   `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

type BuildQueueResponse struct {
	ActiveBuilds        int64   `json:"active_builds"`
	QueuedBuilds        int64   `json:"queued_builds"`
	MaxConcurrentBuilds int64   `json:"max_concurrent_builds"`
	UtilizationPercent  float64 `json:"utilization_percent"`
	Saturated           bool    `json:"saturated"`
}

type DeploymentHealthResponse struct {
	WindowHours            int     `json:"window_hours"`
	Total                  int64   `json:"total"`
	Successful             int64   `json:"successful"`
	Failed                 int64   `json:"failed"`
	Cancelled              int64   `json:"cancelled"`
	SuccessRate            float64 `json:"success_rate"`
	AverageBuildDurationMs *int64  `json:"average_build_duration_ms,omitempty"`
	LastSuccessAt          *string `json:"last_success_at,omitempty"`
	LastFailureAt          *string `json:"last_failure_at,omitempty"`
}

type MetricPointResponse struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type MetricTrendsResponse struct {
	CPUUsage     []MetricPointResponse `json:"cpu_usage"`
	MemoryUsage  []MetricPointResponse `json:"memory_usage"`
	DiskUsage    []MetricPointResponse `json:"disk_usage"`
	QueuedBuilds []MetricPointResponse `json:"queued_builds"`
}

type SystemAlertResponse struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}

type DiskUsageResponse struct {
	DeploymentsBytes int64   `json:"deployments_bytes"`
	DeploymentBytes  int64   `json:"deployment_bytes"`
	LogsBytes        int64   `json:"logs_bytes"`
	DatabaseBytes    int64   `json:"database_bytes"`
	BackupsBytes     int64   `json:"backups_bytes"`
	CacheBytes       int64   `json:"cache_bytes"`
	PlatformBytes    int64   `json:"platform_bytes"`
	TotalBytes       int64   `json:"total_bytes"`
	UsedBytes        int64   `json:"used_bytes"`
	AvailableBytes   int64   `json:"available_bytes"`
	UsagePercent     float64 `json:"usage_percent"`
}

type UserListResponse struct {
	Users      []UserResponse     `json:"users"`
	Pagination PaginationResponse `json:"pagination"`
}

// --- Domain Extras ---

type DNSInstructions struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CreateDomainResponse struct {
	Domain          DomainResponse  `json:"domain"`
	DNSInstructions DNSInstructions `json:"dns_instructions"`
}

// --- Env Var Extras ---

type BulkCreateEnvVarResponse struct {
	EnvVars []EnvVarResponse `json:"env_vars"`
	Created int              `json:"created"`
	Updated int              `json:"updated"`
}
