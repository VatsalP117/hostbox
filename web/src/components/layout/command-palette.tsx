import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { routes } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import {
  LayoutDashboard,
  FolderKanban,
  Plus,
  Shield,
  User,
  Users,
  Activity,
  Settings,
} from "lucide-react";

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, []);

  const runCommand = (path: string) => {
    setOpen(false);
    navigate(path);
  };

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>
        <CommandGroup heading="Navigation">
          <CommandItem onSelect={() => runCommand(routes.dashboard)}>
            <LayoutDashboard className="mr-2 h-4 w-4" />
            Dashboard
          </CommandItem>
          <CommandItem onSelect={() => runCommand(routes.projects)}>
            <FolderKanban className="mr-2 h-4 w-4" />
            Projects
          </CommandItem>
          <CommandItem onSelect={() => runCommand(routes.newProject)}>
            <Plus className="mr-2 h-4 w-4" />
            New Project
          </CommandItem>
          <CommandItem onSelect={() => runCommand(routes.profile)}>
            <User className="mr-2 h-4 w-4" />
            Profile
          </CommandItem>
          {user?.is_admin && (
            <>
              <CommandItem onSelect={() => runCommand(routes.adminTab("overview"))}>
                <Shield className="mr-2 h-4 w-4" />
                System
              </CommandItem>
              <CommandItem onSelect={() => runCommand(routes.adminTab("users"))}>
                <Users className="mr-2 h-4 w-4" />
                Users
              </CommandItem>
              <CommandItem onSelect={() => runCommand(routes.adminTab("activity"))}>
                <Activity className="mr-2 h-4 w-4" />
                Activity
              </CommandItem>
              <CommandItem onSelect={() => runCommand(routes.adminTab("settings"))}>
                <Settings className="mr-2 h-4 w-4" />
                Settings
              </CommandItem>
            </>
          )}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
