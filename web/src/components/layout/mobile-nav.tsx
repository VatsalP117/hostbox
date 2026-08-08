import { Link, useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { routes } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import { BrandMark } from "@/components/shared/brand-mark";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  LayoutDashboard,
  FolderKanban,
  Activity,
  ServerCog,
  Users,
  Settings,
  BookOpen,
  CircleHelp,
  Plus,
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
        className="w-72 border-r border-border bg-black p-0"
      >
        <SheetHeader className="border-b border-border px-4 py-4">
          <SheetTitle className="flex items-center gap-3 text-left">
            <BrandMark className="text-[#52a8ff]" />
            <span className="text-sm font-semibold text-white">Hostbox</span>
          </SheetTitle>
          <SheetDescription className="sr-only">
            Navigate Hostbox projects and administration.
          </SheetDescription>
        </SheetHeader>

        {/* Deploy Button */}
        <div className="border-b border-border px-4 py-4">
          <button
            onClick={() => {
              navigate(routes.newProject);
              onOpenChange(false);
            }}
            className="flex h-9 w-full items-center justify-center gap-2 rounded-md border border-white bg-white text-sm font-medium text-black"
          >
            <Plus className="w-4 h-4" />
            Add project
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
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                isActive(item.path)
                  ? "bg-[#1f1f1f] text-white"
                  : "text-muted-foreground hover:bg-[#171717] hover:text-foreground"
              )}
            >
              <item.icon className="w-5 h-5 shrink-0" />
              <span>{item.label}</span>
            </Link>
          ))}

          {user?.is_admin &&
            adminItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                onClick={() => onOpenChange(false)}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                  isActive(item.path)
                    ? "bg-[#1f1f1f] text-white"
                    : "text-muted-foreground hover:bg-[#171717] hover:text-foreground"
                )}
              >
                <item.icon className="w-5 h-5 shrink-0" />
                <span>{item.label}</span>
              </Link>
            ))}

          {/* Footer Items */}
          <div className="mt-3 space-y-1 border-t border-border pt-3">
            {footerItems.map((item) => (
              <a
                key={item.href}
                href={item.href}
                target="_blank"
                rel="noreferrer"
                onClick={() => onOpenChange(false)}
                className="flex items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-[#171717] hover:text-foreground"
              >
                <item.icon className="w-4 h-4 shrink-0" />
                <span>{item.label}</span>
              </a>
            ))}
          </div>
        </nav>
      </SheetContent>
    </Sheet>
  );
}
