"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { Loader2, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { useCurrentUser, useCurrentOrg, useAuthStore } from "@/stores/auth";
import { grantApi, organizationApi } from "@/lib/api";
import type { ResourceGrant, OrganizationMember } from "@/lib/api";

interface ShareDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resourceType: "pod" | "runner" | "repository";
  resourceId: string;
}

export function ShareDialog({ open, onOpenChange, resourceType, resourceId }: ShareDialogProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const currentUser = useCurrentUser();

  const [grants, setGrants] = useState<ResourceGrant[]>([]);
  const [members, setMembers] = useState<OrganizationMember[]>([]);
  const [selectedUserId, setSelectedUserId] = useState("");
  const [loading, setLoading] = useState(false);
  const [sharing, setSharing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { dialogProps, confirm } = useConfirmDialog();

  const loadData = useCallback(async () => {
    if (!currentOrg) return;
    setLoading(true);
    setError(null);
    try {
      const [grantsRes, membersRes] = await Promise.all([
        grantApi.list(resourceType, resourceId),
        organizationApi.listMembers(currentOrg.slug),
      ]);
      setGrants(grantsRes.grants || []);
      setMembers(membersRes.members || []);
    } catch {
      setError(t("share.loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [currentOrg, resourceType, resourceId, t]);

  useEffect(() => {
    if (open) loadData();
  }, [open, loadData]);

  const grantedUserIds = new Set(grants.map((g) => g.user_id));
  const availableMembers = members.filter(
    (m) => m.user_id !== currentUser?.id && !grantedUserIds.has(m.user_id)
  );

  const handleShare = async () => {
    if (!selectedUserId || !currentOrg) return;
    setSharing(true);
    setError(null);
    try {
      await grantApi.grant(resourceType, resourceId, parseInt(selectedUserId));
      setSelectedUserId("");
      await loadData();
    } catch {
      setError(t("share.grantFailed"));
    } finally {
      setSharing(false);
    }
  };

  const handleRevoke = async (grantId: number) => {
    const confirmed = await confirm({
      title: t("share.revokeConfirmTitle"),
      description: t("share.revokeConfirmDescription"),
      variant: "destructive",
      confirmText: t("share.revoke"),
      cancelText: t("common.cancel"),
    });
    if (!confirmed) return;
    try {
      await grantApi.revoke(resourceType, resourceId, grantId);
      await loadData();
    } catch {
      setError(t("share.revokeFailed"));
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("share.title")}</DialogTitle>
          </DialogHeader>
          <DialogBody className="space-y-4">
            {/* Grant form */}
            <div className="space-y-3 pb-3 border-b border-border">
              <FormField label={t("share.selectUser")} htmlFor="share-user">
                <select
                  id="share-user"
                  value={selectedUserId}
                  onChange={(e) => setSelectedUserId(e.target.value)}
                  className="w-full border border-border rounded px-3 py-2 bg-background text-sm"
                >
                  <option value="">{t("share.selectUserPlaceholder")}</option>
                  {availableMembers.map((m) => (
                    <option key={m.user_id} value={m.user_id}>
                      {m.user?.name || m.user?.username || m.user?.email}
                    </option>
                  ))}
                </select>
              </FormField>
              <Button onClick={handleShare} disabled={sharing || !selectedUserId} className="w-full" size="sm">
                {sharing ? <Loader2 className="w-3 h-3 mr-1 animate-spin" /> : null}
                {t("share.share")}
              </Button>
            </div>

            {/* Grants list */}
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">{t("share.sharedWith")}</p>
              {loading ? (
                <div className="flex items-center justify-center py-4">
                  <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
                </div>
              ) : grants.length === 0 ? (
                <p className="text-sm text-muted-foreground py-2">{t("share.noShares")}</p>
              ) : (
                grants.map((g) => (
                  <div key={g.id} className="flex items-center justify-between py-2 px-2 rounded hover:bg-muted/50">
                    <div className="flex items-center gap-2 min-w-0">
                      <div className="w-6 h-6 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-medium shrink-0">
                        {(g.user?.name || g.user?.username || "?")[0].toUpperCase()}
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium truncate">{g.user?.name || g.user?.username}</p>
                        <p className="text-xs text-muted-foreground truncate">{g.user?.email}</p>
                      </div>
                    </div>
                    <Button size="sm" variant="ghost" className="text-destructive shrink-0" onClick={() => handleRevoke(g.id)}>
                      <X className="w-3 h-3" />
                    </Button>
                  </div>
                ))
              )}
            </div>

            {error && (
              <div className="p-2 bg-destructive/10 text-destructive rounded text-sm">{error}</div>
            )}
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common.close")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <ConfirmDialog {...dialogProps} />
    </>
  );
}
