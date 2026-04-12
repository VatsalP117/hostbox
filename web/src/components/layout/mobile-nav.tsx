import { Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { routes } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { LayoutDashboard, FolderKanban, Shield, User } from "lucide-react";

interface MobileNavProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const navItems = [
  { label: "Dashboard", icon: LayoutDashboard, path: routes.dashboard },
  { label: "Projects", icon: FolderKanban, path: routes.projects },
  { label: "Profile", icon: User, path: routes.profile },
];

export function MobileNav({ open, onOpenChange }: MobileNavProps) {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);

  const isActive = (path: string) => {
    if (path === "/") return location.pathname === "/";
    return location.pathname.startsWith(path);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="left" className="w-64 p-0">
        <SheetHeader className="border-b px-4 py-3">
          <SheetTitle className="flex items-center gap-2 text-left">
            <div className="flex h-7 w-7 items-center justify-center rounded bg-primary">
              <span className="text-xs font-bold text-primary-foreground">
                H
              </span>
            </div>
            Hostbox
          </SheetTitle>
        </SheetHeader>
        <nav className="space-y-1 p-2">
          {navItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => onOpenChange(false)}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive(item.path)
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              <item.icon className="h-4 w-4" />
              <span>{item.label}</span>
            </Link>
          ))}
          {user?.is_admin && (
            <Link
              to={routes.admin}
              onClick={() => onOpenChange(false)}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive(routes.admin)
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              <Shield className="h-4 w-4" />
              <span>Admin</span>
            </Link>
          )}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
