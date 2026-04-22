import type { DeploymentStatus, Framework } from "@/types/models";

// ─── Route Paths ─────────────────────────────────────

export const routes = {
  setup: "/setup",
  login: "/login",
  forgotPassword: "/forgot-password",
  resetPassword: "/reset-password",
  dashboard: "/",
  projects: "/projects",
  newProject: "/projects/new",
  githubSetup: "/github/setup",
  project: (id: string) => `/projects/${id}`,
  deployment: (projectId: string, deploymentId: string) =>
    `/projects/${projectId}/deployments/${deploymentId}`,
  admin: "/admin",
  adminTab: (tab: "overview" | "users" | "activity" | "settings") =>
    `/admin?tab=${tab}`,
  profile: "/profile",
} as const;

// ─── Status Labels & Colors ─────────────────────────

export const statusConfig: Record<
  DeploymentStatus,
  { label: string; className: string; dotClassName: string }
> = {
  queued: {
    label: "Queued",
    className: "bg-muted text-muted-foreground",
    dotClassName: "bg-muted-foreground",
  },
  building: {
    label: "Building",
    className:
      "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
    dotClassName: "bg-blue-500 animate-pulse-slow",
  },
  ready: {
    label: "Ready",
    className:
      "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
    dotClassName: "bg-green-500",
  },
  failed: {
    label: "Failed",
    className:
      "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
    dotClassName: "bg-red-500",
  },
  cancelled: {
    label: "Cancelled",
    className: "bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400",
    dotClassName: "bg-zinc-500",
  },
};

// ─── Framework Display Names ─────────────────────────

export const frameworkConfig: Record<
  Framework,
  { label: string; icon: string }
> = {
  nextjs: { label: "Next.js", icon: "Hexagon" },
  vite: { label: "Vite", icon: "Zap" },
  cra: { label: "Create React App", icon: "Atom" },
  astro: { label: "Astro", icon: "Rocket" },
  gatsby: { label: "Gatsby", icon: "Circle" },
  nuxt: { label: "Nuxt", icon: "Triangle" },
  sveltekit: { label: "SvelteKit", icon: "Flame" },
  hugo: { label: "Hugo", icon: "FileText" },
  "plain-html": { label: "Static HTML", icon: "Globe" },
  unknown: { label: "Unknown", icon: "HelpCircle" },
};

// ─── Node.js Versions ────────────────────────────────

export const nodeVersions = ["22", "20", "18"] as const;

// ─── Query Keys ──────────────────────────────────────

export const queryKeys = {
  setupStatus: ["setup-status"] as const,
  me: ["me"] as const,
  sessions: ["sessions"] as const,
  projects: (params?: { page?: number; search?: string }) =>
    ["projects", params] as const,
  project: (id: string) => ["project", id] as const,
  deployments: (
    projectId: string,
    params?: { page?: number; status?: string; branch?: string },
  ) => ["deployments", projectId, params] as const,
  deployment: (id: string) => ["deployment", id] as const,
  deploymentLogs: (id: string) => ["deployment-logs", id] as const,
  domains: (projectId: string) => ["domains", projectId] as const,
  envVars: (projectId: string) => ["env-vars", projectId] as const,
  githubStatus: ["github-status"] as const,
  installations: ["github-installations"] as const,
  repos: (installationId: number) =>
    ["github-repos", installationId] as const,
  notifications: (projectId: string) =>
    ["notifications", projectId] as const,
  adminStats: ["admin-stats"] as const,
  adminUsers: ["admin-users"] as const,
  adminActivity: (params?: { page?: number }) =>
    ["admin-activity", params] as const,
  adminSettings: ["admin-settings"] as const,
} as const;
