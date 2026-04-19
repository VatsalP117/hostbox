import { Link, useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { routes } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import {
  LayoutDashboard,
  FolderKanban,
  BarChart3,
  Settings2,
  Users,
  Settings,
  FileText,
  HelpCircle,
  Plus,
} from "lucide-react";

const navItems = [
  { label: "Home", icon: LayoutDashboard, path: routes.dashboard },
  { label: "Projects", icon: FolderKanban, path: routes.projects },
];

const adminItems = [
  { label: "System", icon: Settings2, path: routes.adminTab("overview") },
  { label: "Users", icon: Users, path: routes.adminTab("users") },
  { label: "Activity", icon: BarChart3, path: routes.adminTab("activity") },
  { label: "Settings", icon: Settings, path: routes.adminTab("settings") },
];

const footerItems = [
  { label: "Docs", icon: FileText, href: "https://github.com/VatsalP117/hostbox#readme" },
  { label: "Support", icon: HelpCircle, href: "https://github.com/VatsalP117/hostbox/issues" },
];

export function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  const isActive = (path: string) => {
    if (path.includes("?")) {
      return `${location.pathname}${location.search}` === path;
    }
    if (path === "/") return location.pathname === "/";
    return location.pathname.startsWith(path);
  };

  return (
    <aside className="hidden md:flex fixed left-0 top-0 h-full w-64 flex-col py-6 space-y-2 border-r border-[hsl(var(--outline-variant)/0.15)] bg-[#0e0e0e] z-50">
      {/* Logo Section */}
      <div className="px-6 mb-8 flex items-center gap-3">
        <div className="w-8 h-8 rounded-xl bg-[#ADC6FF] flex items-center justify-center text-[#0e0e0e] font-['Manrope'] font-black text-sm">
          H
        </div>
        <div>
          <div className="font-['Manrope'] font-black text-[#ADC6FF] text-lg leading-tight">
            Hostbox
          </div>
          <div className="font-['Space_Grotesk'] text-[10px] text-[#e5e2e1]/70 uppercase tracking-widest">
            Sovereign Control
          </div>
        </div>
      </div>

      {/* Deploy Button */}
      <div className="px-4 mb-6">
        <button
          onClick={() => navigate(routes.newProject)}
          className="w-full gradient-btn text-sm py-2.5 rounded-xl shadow-[0_4px_14px_0_rgba(173,198,255,0.15)] flex items-center justify-center gap-2 transition-all hover:shadow-[0_4px_20px_0_rgba(173,198,255,0.25)]"
        >
          <Plus className="w-4 h-4" />
          Deploy New
        </button>
      </div>

      {/* Main Navigation */}
      <nav className="flex-1 px-3 space-y-1 overflow-y-auto">
        {navItems.map((item) => (
          <Link
            key={item.path}
            to={item.path}
            className={cn(
              "flex items-center gap-3 px-3 py-2 rounded-lg group transition-all",
              isActive(item.path)
                ? "bg-[#201f1f] text-[#ADC6FF] border-r-2 border-[#ADC6FF]"
                : "text-[#e5e2e1]/50 hover:bg-[#1c1b1b] hover:text-[#e5e2e1]"
            )}
          >
            <item.icon className="w-5 h-5 shrink-0" />
            <span className="font-['Space_Grotesk'] text-xs uppercase tracking-widest">
              {item.label}
            </span>
          </Link>
        ))}

        {user?.is_admin &&
          adminItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-lg group transition-all",
                isActive(item.path)
                  ? "bg-[#201f1f] text-[#ADC6FF] border-r-2 border-[#ADC6FF]"
                  : "text-[#e5e2e1]/50 hover:bg-[#1c1b1b] hover:text-[#e5e2e1]"
              )}
            >
              <item.icon className="w-5 h-5 shrink-0" />
              <span className="font-['Space_Grotesk'] text-xs uppercase tracking-widest">
                {item.label}
              </span>
            </Link>
          ))}
      </nav>

      {/* Footer Navigation */}
      <div className="px-3 pt-4 border-t border-[hsl(var(--outline-variant)/0.15)] space-y-1">
        {footerItems.map((item) => (
          <a
            key={item.href}
            href={item.href}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-[#e5e2e1]/50 hover:bg-[#1c1b1b] hover:text-[#e5e2e1] transition-all"
          >
            <item.icon className="w-4 h-4 shrink-0" />
            <span className="font-['Space_Grotesk'] text-xs uppercase tracking-widest">
              {item.label}
            </span>
          </a>
        ))}
      </div>
    </aside>
  );
}
