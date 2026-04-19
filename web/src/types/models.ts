export type DeploymentStatus =
  | "queued"
  | "building"
  | "ready"
  | "failed"
  | "cancelled";

export type ProjectStatus = "healthy" | "failed" | "building" | "stopped";

export type EnvVarScope = "all" | "production" | "preview";

export type NotificationChannel = "discord" | "slack" | "webhook";

export type NotificationEvent =
  | "deploy_failure"
  | "deploy_success"
  | "domain_verified";

export type Framework =
  | "nextjs"
  | "vite"
  | "cra"
  | "astro"
  | "gatsby"
  | "nuxt"
  | "sveltekit"
  | "hugo"
  | "plain-html"
  | "unknown";

export type ActivityAction =
  | "project.created"
  | "project.updated"
  | "project.deleted"
  | "deployment.created"
  | "deployment.cancelled"
  | "deployment.rolled_back"
  | "domain.added"
  | "domain.verified"
  | "domain.deleted"
  | "env_var.created"
  | "env_var.updated"
  | "env_var.deleted"
  | "user.login"
  | "user.logout"
  | "user.created"
  | "settings.updated";

export type ResourceType =
  | "project"
  | "deployment"
  | "domain"
  | "env_var"
  | "user"
  | "settings";

export interface User {
  id: string;
  email: string;
  display_name: string;
  is_admin: boolean;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  owner_id: string;
  name: string;
  slug: string;
  github_repo: string | null;
  github_installation_id: number | null;
  production_branch: string;
  framework: Framework | null;
  build_command: string | null;
  install_command: string | null;
  output_directory: string | null;
  root_directory: string;
  node_version: string;
  auto_deploy: boolean;
  preview_deployments: boolean;
  status?: ProjectStatus;
  created_at: string;
  updated_at: string;
}

export interface ProjectStats {
  total_deployments: number;
  successful_builds: number;
  failed_builds: number;
  average_build_time_ms: number | null;
  last_deploy_at: string | null;
}

export interface Deployment {
  id: string;
  project_id: string;
  commit_sha: string;
  commit_message: string | null;
  commit_author: string | null;
  branch: string;
  status: DeploymentStatus;
  is_production: boolean;
  deployment_url: string | null;
  artifact_path: string;
  artifact_size_bytes: number | null;
  log_path: string;
  error_message: string | null;
  is_rollback: boolean;
  rollback_source_id: string | null;
  github_pr_number: number | null;
  build_duration_ms: number | null;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface Domain {
  id: string;
  project_id: string;
  domain: string;
  verified: boolean;
  verified_at: string | null;
  last_checked_at: string | null;
  created_at: string;
}

export interface EnvVar {
  id: string;
  project_id: string;
  key: string;
  value: string;
  is_secret: boolean;
  scope: EnvVarScope;
  created_at: string;
  updated_at: string;
}

export interface NotificationConfig {
  id: string;
  project_id: string | null;
  channel: NotificationChannel;
  webhook_url: string;
  events: string;
  enabled: boolean;
  created_at: string;
}

export interface GitHubRepo {
  name: string;
  full_name: string;
  html_url: string;
  description: string | null;
  private: boolean;
  default_branch: string;
  language: string | null;
}

export interface GitHubInstallation {
  id: number;
  account: string;
  avatar_url: string;
  target_type: "User" | "Organization";
}

export interface Session {
  id: string;
  user_agent: string;
  ip_address: string;
  is_current: boolean;
  last_active_at: string;
  expires_at: string;
  created_at: string;
}

export interface Activity {
  id: number;
  user_id: string | null;
  user_name?: string | null;
  action: string;
  resource_type: string;
  resource_id: string | null;
  project_name?: string | null;
  metadata: string | null;
  created_at: string;
}

export interface SystemStats {
  components: {
    api: ServiceHealth;
    database: ServiceHealth;
    docker: ServiceHealth;
    caddy: ServiceHealth;
  };
  cpu: {
    usage_percent: number;
    cores: number;
    load1: number;
    load5: number;
    load15: number;
  };
  memory: {
    used_bytes: number;
    total_bytes: number;
    available_bytes: number;
    usage_percent: number;
  };
  build_queue: {
    active_builds: number;
    queued_builds: number;
    max_concurrent_builds: number;
    utilization_percent: number;
    saturated: boolean;
  };
  deployment_health: {
    window_hours: number;
    total: number;
    successful: number;
    failed: number;
    cancelled: number;
    success_rate: number;
    average_build_duration_ms: number | null;
    last_success_at: string | null;
    last_failure_at: string | null;
  };
  trends: {
    cpu_usage: MetricPoint[];
    memory_usage: MetricPoint[];
    disk_usage: MetricPoint[];
    queued_builds: MetricPoint[];
  };
  alerts: SystemAlert[];
  disk_usage: {
    total_bytes: number;
    used_bytes: number;
    available_bytes: number;
    deployment_bytes: number;
    deployments_bytes: number;
    logs_bytes: number;
    database_bytes: number;
    backups_bytes: number;
    cache_bytes: number;
    platform_bytes: number;
    usage_percent: number;
  };
  project_count: number;
  deployment_count: number;
  active_builds: number;
  queued_builds: number;
  max_concurrent_builds: number;
  user_count?: number;
  uptime_seconds: number;
}

export interface ServiceHealth {
  status: "healthy" | "degraded" | "unavailable" | string;
  message?: string;
}

export interface MetricPoint {
  timestamp: string;
  value: number;
}

export interface SystemAlert {
  severity: "info" | "warning" | "error" | string;
  title: string;
  message: string;
}

export interface PlatformSettings {
  registration_enabled: boolean;
  max_projects: number;
  max_concurrent_builds: number;
  artifact_retention_days: number;
}

export interface DnsInstructions {
  a_record: string;
  cname_record: string;
}
