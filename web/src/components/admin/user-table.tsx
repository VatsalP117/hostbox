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
      <div className="bg-surface-container-low rounded-xl p-6 space-y-4">
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
      <div className="bg-surface-container-low rounded-xl p-8 text-center">
        <p className="font-body text-muted-foreground">No users found</p>
      </div>
    );
  }

  return (
    <div className="bg-surface-container-low rounded-xl overflow-hidden">
      <div className="p-6 border-b border-outline-variant/15">
        <h3 className="font-headline text-lg font-bold text-foreground">Platform Users</h3>
        <p className="font-label text-xs text-muted-foreground mt-1 uppercase tracking-wider">
          {users.length} total users
        </p>
      </div>
      <Table>
        <TableHeader>
          <TableRow className="border-outline-variant/15 hover:bg-transparent">
            <TableHead className="font-label text-xs uppercase tracking-wider text-muted-foreground">User</TableHead>
            <TableHead className="font-label text-xs uppercase tracking-wider text-muted-foreground">Email</TableHead>
            <TableHead className="font-label text-xs uppercase tracking-wider text-muted-foreground">Role</TableHead>
              <TableHead className="font-label text-xs uppercase tracking-wider text-muted-foreground">Joined</TableHead>
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
                    <span className="font-headline text-sm text-foreground">
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
                    className="gap-1 bg-primary/10 text-primary border-primary/30 font-label text-xs uppercase tracking-wider"
                  >
                    <Shield className="h-3 w-3" />
                    Admin
                  </Badge>
                ) : (
                  <Badge 
                    variant="secondary" 
                    className="font-label text-xs uppercase tracking-wider bg-surface-container-high text-muted-foreground"
                  >
                    User
                  </Badge>
                )}
              </TableCell>
              <TableCell>
                <span className="font-label text-xs text-muted-foreground">
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
