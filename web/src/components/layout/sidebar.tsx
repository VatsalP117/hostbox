import { Link, useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { routes } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import { BrandMark } from "@/components/shared/brand-mark";
import {
  Activity,
  BookOpen,
  Boxes,
  CircleHelp,
  FolderKanban,
  LayoutDashboard,
  Plus,
  ServerCog,
  Settings,
  Users,
} from "lucide-react";

const navItems = [
  { label: "Overview", icon: LayoutDashboard, path: routes.dashboard },
  { label: "Projects", icon: FolderKanban, path: routes.projects },
];

const adminItems = [
  { label: "System", icon: ServerCog, path: routes.adminTab("overview") },
  { label: "Users", icon: Users, path: routes.adminTab("users") },
  { label: "Activity", icon: Activity, path: routes.adminTab("activity") },
  { label: "Settings", icon: Settings, path: routes.adminTab("settings") },
];

const footerItems = [
  { label: "Documentation", icon: BookOpen, href: "https://github.com/VatsalP117/hostbox#readme" },
  { label: "Support", icon: CircleHelp, href: "https://github.com/VatsalP117/hostbox/issues" },
];

export function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  const isActive = (path: string) => {
    if (path.includes("?")) return `${location.pathname}${location.search}` === path;
    if (path === "/") return location.pathname === "/";
    return location.pathname.startsWith(path);
  };

  const itemClass = (path: string) =>
    cn(
      "flex h-9 items-center gap-3 rounded-md px-3 text-sm transition-colors",
      isActive(path)
        ? "bg-[#1f1f1f] text-white"
        : "text-muted-foreground hover:bg-[#171717] hover:text-foreground",
    );

  return (
    <aside className="fixed inset-y-0 left-0 z-50 hidden w-60 flex-col border-r border-border bg-black md:flex">
      <div className="flex h-14 items-center border-b border-border px-4">
        <Link to={routes.dashboard} className="flex items-center gap-2.5" aria-label="Hostbox dashboard">
          <BrandMark className="text-[#52a8ff]" />
          <span className="text-sm font-semibold tracking-tight text-white">Hostbox</span>
        </Link>
        <span className="ml-auto rounded-full border border-border px-2 py-0.5 text-[10px] text-muted-foreground">Self-hosted</span>
      </div>

      <div className="p-3">
        <button
          onClick={() => navigate(routes.newProject)}
          className="flex h-9 w-full items-center justify-center gap-2 rounded-md border border-white bg-white text-sm font-medium text-black transition-colors hover:bg-neutral-200"
        >
          <Plus className="h-4 w-4" />
          Add project
        </button>
      </div>

      <nav className="flex-1 space-y-1 overflow-y-auto px-3 pb-4">
        {navItems.map((item) => (
          <Link key={item.path} to={item.path} className={itemClass(item.path)}>
            <item.icon className="h-4 w-4" />
            {item.label}
          </Link>
        ))}

        {user?.is_admin && (
          <div className="pt-5">
            <p className="mb-2 px-3 text-xs font-medium text-muted-foreground">Administration</p>
            <div className="space-y-1">
              {adminItems.map((item) => (
                <Link key={item.path} to={item.path} className={itemClass(item.path)}>
                  <item.icon className="h-4 w-4" />
                  {item.label}
                </Link>
              ))}
            </div>
          </div>
        )}
      </nav>

      <div className="border-t border-border p-3">
        <div className="mb-2 flex items-center gap-2 rounded-md px-3 py-2 text-xs text-muted-foreground">
          <Boxes className="h-4 w-4 text-foreground" />
          <span className="truncate">Your infrastructure</span>
          <span className="ml-auto h-1.5 w-1.5 rounded-full bg-success" title="Online" />
        </div>
        {footerItems.map((item) => (
          <a key={item.href} href={item.href} target="_blank" rel="noreferrer" className="flex h-8 items-center gap-3 rounded-md px-3 text-xs text-muted-foreground transition-colors hover:bg-[#171717] hover:text-foreground">
            <item.icon className="h-3.5 w-3.5" />
            {item.label}
          </a>
        ))}
      </div>
    </aside>
  );
}
