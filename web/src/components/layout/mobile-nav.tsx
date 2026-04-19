import { Link, useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { routes } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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

interface MobileNavProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MobileNav({ open, onOpenChange }: MobileNavProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  const isActive = (path: string) => {
    if (path.includes("?")) return `${location.pathname}${location.search}` === path;
    if (path === "/") return location.pathname === "/";
    return location.pathname.startsWith(path);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="left"
        className="w-72 p-0 bg-[#0e0e0e] border-r border-[hsl(var(--outline-variant)/0.15)]"
      >
        <SheetHeader className="border-b border-[hsl(var(--outline-variant)/0.15)] px-4 py-5">
          <SheetTitle className="flex items-center gap-3 text-left">
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
          </SheetTitle>
        </SheetHeader>

        {/* Deploy Button */}
        <div className="px-4 py-4 border-b border-[hsl(var(--outline-variant)/0.15)]">
          <button
            onClick={() => {
              navigate(routes.newProject);
              onOpenChange(false);
            }}
            className="w-full gradient-btn text-sm py-2.5 rounded-xl shadow-[0_4px_14px_0_rgba(173,198,255,0.15)] flex items-center justify-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Deploy New
          </button>
        </div>

        {/* Navigation */}
        <nav className="space-y-1 p-3 overflow-y-auto max-h-[calc(100vh-240px)]">
          {navItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => onOpenChange(false)}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2.5 transition-colors",
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
                onClick={() => onOpenChange(false)}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2.5 transition-colors",
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

          {/* Footer Items */}
          <div className="pt-3 mt-3 border-t border-[hsl(var(--outline-variant)/0.15)] space-y-1">
            {footerItems.map((item) => (
              <a
                key={item.href}
                href={item.href}
                target="_blank"
                rel="noreferrer"
                onClick={() => onOpenChange(false)}
                className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-[#e5e2e1]/50 hover:bg-[#1c1b1b] hover:text-[#e5e2e1] transition-colors"
              >
                <item.icon className="w-4 h-4 shrink-0" />
                <span className="font-['Space_Grotesk'] text-xs uppercase tracking-widest">
                  {item.label}
                </span>
              </a>
            ))}
          </div>
        </nav>
      </SheetContent>
    </Sheet>
  );
}
