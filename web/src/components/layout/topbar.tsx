import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { useLogout } from "@/hooks/use-auth";
import { BrandMark } from "@/components/shared/brand-mark";
import { routes } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Bell, Menu, User, LogOut, Search, Plus } from "lucide-react";

interface TopbarProps {
  onMobileMenuToggle: () => void;
}

export function Topbar({ onMobileMenuToggle }: TopbarProps) {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useLogout();

  const initials = user?.display_name
    ? user.display_name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "U";

  const handleLogout = () => {
    logout.mutate(undefined, {
      onSettled: () => navigate(routes.login),
    });
  };

  return (
    <header className="z-40 flex h-14 shrink-0 items-center justify-between border-b border-border bg-black/90 px-4 backdrop-blur-xl md:hidden">
      {/* Mobile: Logo */}
      <div className="flex items-center gap-2 text-sm font-semibold text-white">
        <BrandMark className="h-5 w-5 text-[#52a8ff]" />
        Hostbox
      </div>

      {/* Mobile: Hamburger Menu */}
      <Button
        variant="ghost"
        size="icon"
        className="h-9 w-9 text-foreground hover:bg-accent"
        onClick={onMobileMenuToggle}
        aria-label="Open navigation"
      >
        <Menu className="h-6 w-6" />
      </Button>
    </header>
  );
}

export function DesktopTopbar() {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useLogout();

  const initials = user?.display_name
    ? user.display_name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "U";

  const handleLogout = () => {
    logout.mutate(undefined, {
      onSettled: () => navigate(routes.login),
    });
  };

  return (
    <header className="hidden h-14 items-center justify-between border-b border-border bg-background/90 px-6 backdrop-blur-xl md:flex">
      {/* Search */}
      <div className="flex flex-1 items-center gap-2">
        <Button
          variant="outline"
          className="hidden h-8 w-64 justify-start border-border bg-black text-xs text-muted-foreground hover:bg-accent md:flex"
          onClick={() =>
            document.dispatchEvent(
              new KeyboardEvent("keydown", { key: "k", metaKey: true }),
            )
          }
        >
          <Search className="mr-2 h-4 w-4" />
          Find anything...
          <kbd className="ml-auto rounded border border-border bg-[#171717] px-1.5 py-0.5 text-[10px] font-medium">
            ⌘K
          </kbd>
        </Button>
      </div>

      {/* User Menu */}
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground" aria-label="Notifications">
          <Bell className="h-4 w-4" />
        </Button>
        <Button variant="outline" size="sm" className="mr-2 h-8 bg-black" onClick={() => navigate(routes.newProject)}>
          <Plus className="h-3.5 w-3.5" />
          Add new
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="relative h-8 w-8 rounded-full hover:bg-accent"
            >
              <Avatar className="h-8 w-8 border border-border bg-[#171717]">
                <AvatarFallback className="bg-transparent text-xs text-foreground">
                  {initials}
                </AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            className="w-56 border-border bg-[#111]"
          >
            <div className="flex items-center gap-2 p-2">
              <div className="flex flex-col space-y-0.5">
                <p className="text-sm font-medium text-foreground">
                  {user?.display_name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {user?.email}
                </p>
              </div>
            </div>
            <DropdownMenuSeparator className="bg-border" />
            <DropdownMenuItem
              onClick={() => navigate(routes.profile)}
              className="cursor-pointer text-foreground focus:bg-accent"
            >
              <User className="mr-2 h-4 w-4" />
              Profile
            </DropdownMenuItem>
            <DropdownMenuSeparator className="bg-border" />
            <DropdownMenuItem
              onClick={handleLogout}
              className="cursor-pointer text-foreground focus:bg-accent"
            >
              <LogOut className="mr-2 h-4 w-4" />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
