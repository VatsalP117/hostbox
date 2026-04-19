import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { useLogout } from "@/hooks/use-auth";
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
import { Menu, User, LogOut, Search } from "lucide-react";

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
    <header className="md:hidden flex items-center justify-between px-6 h-16 bg-[#131313]/70 backdrop-blur-xl border-b border-[hsl(var(--outline-variant)/0.15)] z-40 shrink-0">
      {/* Mobile: Logo */}
      <div className="font-['Manrope'] font-black text-[#ADC6FF] text-lg">
        Hostbox
      </div>

      {/* Mobile: Hamburger Menu */}
      <Button
        variant="ghost"
        size="icon"
        className="h-10 w-10 text-[#e5e2e1] hover:bg-[#1c1b1b]"
        onClick={onMobileMenuToggle}
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
    <header className="hidden md:flex h-16 items-center justify-between px-6 border-b border-[hsl(var(--outline-variant)/0.15)] bg-[#131313]">
      {/* Search */}
      <div className="flex flex-1 items-center gap-2">
        <Button
          variant="outline"
          className="hidden h-9 w-72 justify-start text-sm text-[#e5e2e1]/50 bg-transparent border-[hsl(var(--outline-variant)/0.3)] hover:bg-[#1c1b1b] md:flex"
          onClick={() =>
            document.dispatchEvent(
              new KeyboardEvent("keydown", { key: "k", metaKey: true }),
            )
          }
        >
          <Search className="mr-2 h-4 w-4" />
          Search...
          <kbd className="ml-auto rounded border border-[hsl(var(--outline-variant)/0.3)] bg-[#1c1b1b] px-1.5 py-0.5 text-[10px] font-medium font-['Space_Grotesk']">
            ⌘K
          </kbd>
        </Button>
      </div>

      {/* User Menu */}
      <div className="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="relative h-9 w-9 rounded-full hover:bg-[#1c1b1b]"
            >
              <Avatar className="h-9 w-9 bg-[#201f1f] border border-[hsl(var(--outline-variant)/0.3)]">
                <AvatarFallback className="text-xs font-['Space_Grotesk'] text-[#ADC6FF] bg-transparent">
                  {initials}
                </AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            className="w-56 bg-[#1a1a1a] border-[hsl(var(--outline-variant)/0.3)]"
          >
            <div className="flex items-center gap-2 p-2">
              <div className="flex flex-col space-y-0.5">
                <p className="text-sm font-medium text-[#e5e2e1] font-['Manrope']">
                  {user?.display_name}
                </p>
                <p className="text-xs text-[#e5e2e1]/50 font-['Space_Grotesk']">
                  {user?.email}
                </p>
              </div>
            </div>
            <DropdownMenuSeparator className="bg-[hsl(var(--outline-variant)/0.3)]" />
            <DropdownMenuItem
              onClick={() => navigate(routes.profile)}
              className="text-[#e5e2e1] hover:bg-[#201f1f] focus:bg-[#201f1f] cursor-pointer"
            >
              <User className="mr-2 h-4 w-4 text-[#ADC6FF]" />
              Profile
            </DropdownMenuItem>
            <DropdownMenuSeparator className="bg-[hsl(var(--outline-variant)/0.3)]" />
            <DropdownMenuItem
              onClick={handleLogout}
              className="text-[#e5e2e1] hover:bg-[#201f1f] focus:bg-[#201f1f] cursor-pointer"
            >
              <LogOut className="mr-2 h-4 w-4 text-[#ADC6FF]" />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
