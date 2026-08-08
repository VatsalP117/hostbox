import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { TimeAgo } from "@/components/shared/time-ago";
import type { User } from "@/types/models";
import { Shield } from "lucide-react";

interface UserTableProps {
  users: User[] | undefined;
  isLoading: boolean;
}

export function UserTable({ users, isLoading }: UserTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-4 rounded-lg border border-border bg-card p-6">
        <Skeleton className="h-8 w-48 bg-surface-container" />
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-lg bg-surface-container" />
          ))}
        </div>
      </div>
    );
  }

  if (!users?.length) {
    return (
      <div className="rounded-lg border border-dashed border-border bg-card/50 p-8 text-center">
        <p className="font-body text-muted-foreground">No users found</p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="p-6 border-b border-outline-variant/15">
        <h3 className="text-base font-medium text-foreground">Platform users</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          {users.length} total users
        </p>
      </div>
      <Table>
        <TableHeader>
          <TableRow className="border-outline-variant/15 hover:bg-transparent">
            <TableHead className="text-xs text-muted-foreground">User</TableHead>
            <TableHead className="text-xs text-muted-foreground">Email</TableHead>
            <TableHead className="text-xs text-muted-foreground">Role</TableHead>
              <TableHead className="text-xs text-muted-foreground">Joined</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((user) => (
            <TableRow 
              key={user.id} 
              className="border-outline-variant/15 hover:bg-surface-container transition-colors"
            >
              <TableCell>
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 rounded-full bg-surface-container-high flex items-center justify-center">
                    <span className="font-sans text-sm text-foreground">
                      {user.display_name.charAt(0).toUpperCase()}
                    </span>
                  </div>
                  <span className="font-body text-sm font-medium text-foreground">
                    {user.display_name}
                  </span>
                </div>
              </TableCell>
              <TableCell className="font-body text-sm text-muted-foreground">
                {user.email}
              </TableCell>
              <TableCell>
                {user.is_admin ? (
                  <Badge 
                    variant="default" 
                    className="gap-1 bg-primary/10 text-primary border-primary/30 text-xs"
                  >
                    <Shield className="h-3 w-3" />
                    Admin
                  </Badge>
                ) : (
                  <Badge 
                    variant="secondary" 
                    className="text-xs bg-surface-container-high text-muted-foreground"
                  >
                    User
                  </Badge>
                )}
              </TableCell>
              <TableCell>
                <span className="font-sans text-xs text-muted-foreground">
                  <TimeAgo date={user.created_at} />
                </span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
