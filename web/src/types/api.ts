import type {
  User,
  Project,
  Deployment,
  Domain,
  EnvVar,
  NotificationConfig,
  GitHubRepo,
  GitHubInstallation,
  ProjectStats,
  SystemStats,
  PlatformSettings,
  Activity,
  Session,
  DnsInstructions,
  DeploymentStatus,
  EnvVarScope,
  NotificationChannel,
} from "./models";

// ─── Pagination ──────────────────────────────────────

export interface Pagination {
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

// ─── Error ───────────────────────────────────────────

export interface ApiErrorDetail {
  field: string;
  message: string;
}

export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: ApiErrorDetail[];
  };
}

// ─── Setup ───────────────────────────────────────────

export interface SetupStatusResponse {
  setup_required: boolean;
}

export interface SetupRequest {
  email: string;
  password: string;
  display_name: string;
}

export interface SetupResponse {
  user: User;
  access_token: string;
}

// ─── Auth ────────────────────────────────────────────

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  access_token: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  display_name?: string;
}

export interface RegisterResponse {
  user: User;
  access_token: string;
}

export interface RefreshResponse {
  access_token: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  token: string;
  new_password: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export interface UpdateProfileRequest {
  display_name?: string;
  email?: string;
  current_password?: string;
}

export interface MeResponse {
  user: User;
}

export interface LogoutAllResponse {
  success: boolean;
  sessions_revoked: number;
}

// ─── Projects ────────────────────────────────────────

export interface CreateProjectRequest {
  name: string;
  github_repo?: string;
  github_installation_id?: number;
  build_command?: string;
  install_command?: string;
  output_directory?: string;
  root_directory?: string;
  node_version?: string;
}

export interface UpdateProjectRequest {
  name?: string;
  build_command?: string;
  install_command?: string;
  output_directory?: string;
  root_directory?: string;
  node_version?: string;
  production_branch?: string;
  auto_deploy?: boolean;
  preview_deployments?: boolean;
}

export interface ProjectListResponse {
  projects: Project[];
  pagination: Pagination;
}

export interface ProjectDetailResponse {
  project: Project;
  latest_deployment: Deployment | null;
  domains: Domain[];
  stats: ProjectStats;
  env_vars_count: number;
  notifications_count: number;
}

// ─── Deployments ─────────────────────────────────────

export interface TriggerDeployRequest {
  branch: string;
  commit_sha?: string;
  commit_message?: string;
  commit_author?: string;
}

export interface DeploymentListResponse {
  deployments: Deployment[];
  pagination: Pagination;
}

export interface DeploymentResponse {
  deployment: Deployment;
}

export interface LogsResponse {
  lines: string[];
  total_lines: number;
  has_more: boolean;
}

// ─── Domains ─────────────────────────────────────────

export interface CreateDomainRequest {
  domain: string;
}

export interface CreateDomainResponse {
  domain: Domain;
  dns_instructions: DnsInstructions;
}

export interface DomainListResponse {
  domains: Domain[];
}

export interface VerifyDomainResponse {
  domain: Domain;
}

// ─── Environment Variables ───────────────────────────

export interface CreateEnvVarRequest {
  key: string;
  value: string;
  scope?: EnvVarScope;
}

export interface UpdateEnvVarRequest {
  value?: string;
  scope?: EnvVarScope;
}

export interface BulkImportEnvVarRequest {
  env_vars: Array<{ key: string; value: string }>;
  scope?: EnvVarScope;
}

export interface EnvVarListResponse {
  env_vars: EnvVar[];
}

export interface BulkImportEnvVarResponse {
  env_vars: EnvVar[];
  created: number;
  updated: number;
}

// ─── GitHub ──────────────────────────────────────────

export interface GitHubStatusResponse {
  configured: boolean;
  app_slug?: string;
  install_url?: string;
}

export interface GitHubManifestResponse {
  action_url: string;
  manifest: Record<string, unknown>;
}

export interface CompleteGitHubManifestRequest {
  code: string;
  state: string;
}

export interface GitHubReposResponse {
  repos: GitHubRepo[];
  pagination: Pagination;
}

export interface GitHubInstallationsResponse {
  installations: GitHubInstallation[];
}

// ─── Notifications ───────────────────────────────────

export interface CreateNotificationRequest {
  channel: NotificationChannel;
  webhook_url: string;
  events?: string;
}

export interface UpdateNotificationRequest {
  webhook_url?: string;
  events?: string;
  enabled?: boolean;
}

export interface NotificationListResponse {
  notifications: NotificationConfig[];
}

// ─── Admin ───────────────────────────────────────────

export type AdminStatsResponse = SystemStats;

export interface AdminUsersResponse {
  users: User[];
}

export interface AdminActivityResponse {
  activities: Activity[];
  pagination: Pagination;
}

export interface UpdateSettingsRequest {
  registration_enabled?: boolean;
  max_projects?: number;
  max_concurrent_builds?: number;
  artifact_retention_days?: number;
}

export interface AdminSettingsResponse {
  settings: PlatformSettings;
}

// ─── User Profile ────────────────────────────────────

export interface SessionsResponse {
  sessions: Session[];
}

// ─── Health ──────────────────────────────────────────

export interface HealthResponse {
  status: "ok";
  version: string;
  uptime_seconds: number;
}

// ─── Query Params ────────────────────────────────────

export interface PaginationParams {
  page?: number;
  per_page?: number;
}

export interface DeploymentListParams extends PaginationParams {
  status?: DeploymentStatus;
  branch?: string;
}

export interface ProjectListParams extends PaginationParams {
  search?: string;
}

export interface AdminActivityParams extends PaginationParams {
  action?: string;
  resource_type?: string;
}

export interface GitHubReposParams extends PaginationParams {
  installation_id: number;
}
