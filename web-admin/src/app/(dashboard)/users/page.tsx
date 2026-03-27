"use client";

import { useState, useEffect, useCallback } from "react";
import { Search } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  listUsers,
  disableUser,
  enableUser,
  grantAdmin,
  revokeAdmin,
  verifyUserEmail,
  unverifyUserEmail,
  User,
} from "@/lib/api/admin";
import type { PaginatedResponse } from "@/lib/api/base";
import { UserRow } from "./user-row";

export default function UsersPage() {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<PaginatedResponse<User> | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchUsers = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await listUsers({ search, page, page_size: 20 });
      setData(result);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, [search, page]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  const makeHandler = (action: (id: number) => Promise<unknown>, successMsg: string, errorMsg: string) => {
    return async (userId: number) => {
      try {
        await action(userId);
        toast.success(successMsg);
        await fetchUsers();
      } catch (err: unknown) {
        toast.error((err as { error?: string })?.error || errorMsg);
      }
    };
  };

  const handleDisable = makeHandler(disableUser, "User disabled successfully", "Failed to disable user");
  const handleEnable = makeHandler(enableUser, "User enabled successfully", "Failed to enable user");
  const handleGrantAdmin = makeHandler(grantAdmin, "Admin privileges granted", "Failed to grant admin privileges");
  const handleRevokeAdmin = makeHandler(revokeAdmin, "Admin privileges revoked", "Failed to revoke admin privileges");
  const handleVerifyEmail = makeHandler(verifyUserEmail, "Email verified successfully", "Failed to verify email");
  const handleUnverifyEmail = makeHandler(unverifyUserEmail, "Email unverified successfully", "Failed to unverify email");

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <div className="relative flex-1 sm:max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search users..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Users ({data?.total || 0})</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-16 animate-pulse rounded-lg bg-muted" />
              ))}
            </div>
          ) : (
            <div className="space-y-2">
              {data?.data.map((user) => (
                <UserRow
                  key={user.id}
                  user={user}
                  onDisable={() => handleDisable(user.id)}
                  onEnable={() => handleEnable(user.id)}
                  onGrantAdmin={() => handleGrantAdmin(user.id)}
                  onRevokeAdmin={() => handleRevokeAdmin(user.id)}
                  onVerifyEmail={() => handleVerifyEmail(user.id)}
                  onUnverifyEmail={() => handleUnverifyEmail(user.id)}
                />
              ))}
              {data?.data.length === 0 && (
                <p className="py-8 text-center text-muted-foreground">No users found</p>
              )}
            </div>
          )}

          {data && data.total_pages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Page {data.page} of {data.total_pages}
              </p>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage(page - 1)}>
                  Previous
                </Button>
                <Button variant="outline" size="sm" disabled={page >= data.total_pages} onClick={() => setPage(page + 1)}>
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
