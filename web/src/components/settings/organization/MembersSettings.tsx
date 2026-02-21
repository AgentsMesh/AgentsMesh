"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { FormField } from "@/components/ui/form-field";
import { useAuthStore } from "@/stores/auth";
import { organizationApi } from "@/lib/api";
import { ApiError } from "@/lib/api/base";
import { invitationApi, type Invitation } from "@/lib/api/invitation";
import type { TranslationFn } from "./GeneralSettings";

interface Member {
  id: number;
  user_id: number;
  role: string;
  joined_at: string;
  user?: { id: number; email: string; username: string; name?: string };
}

interface MembersSettingsProps {
  t: TranslationFn;
}

export function MembersSettings({ t }: MembersSettingsProps) {
  const router = useRouter();
  const { currentOrg, user } = useAuthStore();
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"admin" | "member">("member");
  const [inviting, setInviting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [errorType, setErrorType] = useState<"generic" | "no_seats" | "subscription_frozen">("generic");
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Pending invitations state
  const [pendingInvitations, setPendingInvitations] = useState<Invitation[]>([]);
  const [loadingInvitations, setLoadingInvitations] = useState(false);
  const [resendingId, setResendingId] = useState<number | null>(null);

  const removeMemberDialog = useConfirmDialog({
    title: t("settings.members.removeDialog.title"),
    description: t("settings.members.removeDialog.description"),
    confirmText: t("settings.members.remove"),
    variant: "destructive",
  });

  const revokeInvitationDialog = useConfirmDialog({
    title: t("settings.members.revokeDialog.title"),
    description: t("settings.members.revokeDialog.description"),
    confirmText: t("settings.members.revoke"),
    variant: "destructive",
  });

  const loadMembers = useCallback(async () => {
    if (!currentOrg) return;
    try {
      setLoading(true);
      const response = await organizationApi.listMembers(currentOrg.slug);
      setMembers(response.members || []);
    } catch (err) {
      console.error("Failed to load members:", err);
      setError(t("settings.members.failedToLoad"));
    } finally {
      setLoading(false);
    }
  }, [currentOrg, t]);

  const loadInvitations = useCallback(async () => {
    if (!currentOrg) return;
    try {
      setLoadingInvitations(true);
      const response = await invitationApi.list();
      setPendingInvitations(response.invitations || []);
    } catch (err) {
      console.error("Failed to load invitations:", err);
    } finally {
      setLoadingInvitations(false);
    }
  }, [currentOrg]);

  useEffect(() => {
    loadMembers();
    loadInvitations();
  }, [loadMembers, loadInvitations]);

  // Auto-dismiss success message
  useEffect(() => {
    if (!successMessage) return;
    const timer = setTimeout(() => setSuccessMessage(null), 5000);
    return () => clearTimeout(timer);
  }, [successMessage]);

  const handleInvite = async () => {
    if (!currentOrg || !inviteEmail) return;
    setInviting(true);
    setError(null);
    try {
      await invitationApi.create(inviteEmail, inviteRole);
      setShowInviteDialog(false);
      setInviteEmail("");
      setInviteRole("member");
      setSuccessMessage(t("settings.members.inviteSent", { email: inviteEmail }));
      await loadInvitations();
    } catch (err) {
      console.error("Failed to invite member:", err);
      setErrorType("generic");

      if (err instanceof ApiError && err.data) {
        const data = err.data as { code?: string; error?: string };
        const code = data.code || "";
        const errorStr = data.error || "";

        if (code === "NO_AVAILABLE_SEATS") {
          setError(t("settings.members.noSeats"));
          setErrorType("no_seats");
        } else if (code === "SUBSCRIPTION_FROZEN") {
          setError(t("settings.members.subscriptionFrozen"));
          setErrorType("subscription_frozen");
        } else if (errorStr.includes("already a member")) {
          setError(t("settings.members.alreadyMember"));
        } else if (errorStr.includes("pending invitation already exists")) {
          setError(t("settings.members.pendingExists"));
        } else {
          setError(t("settings.members.failedToInvite"));
        }
      } else {
        setError(t("settings.members.failedToInvite"));
      }
    } finally {
      setInviting(false);
    }
  };

  const handleRevoke = async (invitationId: number) => {
    const confirmed = await revokeInvitationDialog.confirm();
    if (!confirmed) return;
    try {
      await invitationApi.revoke(invitationId);
      setSuccessMessage(t("settings.members.invitationRevoked"));
      await loadInvitations();
    } catch (err) {
      console.error("Failed to revoke invitation:", err);
      setError(t("settings.members.failedToRevoke"));
    }
  };

  const handleResend = async (invitationId: number) => {
    setResendingId(invitationId);
    try {
      await invitationApi.resend(invitationId);
      setSuccessMessage(t("settings.members.invitationResent"));
    } catch (err) {
      console.error("Failed to resend invitation:", err);
      setError(t("settings.members.failedToResend"));
    } finally {
      setResendingId(null);
    }
  };

  const handleRemove = async (userId: number) => {
    if (!currentOrg) return;
    const confirmed = await removeMemberDialog.confirm();
    if (!confirmed) return;
    try {
      await organizationApi.removeMember(currentOrg.slug, userId);
      await loadMembers();
    } catch (err) {
      console.error("Failed to remove member:", err);
      setError(t("settings.members.failedToRemove"));
    }
  };

  const handleRoleChange = async (userId: number, newRole: string) => {
    if (!currentOrg) return;
    try {
      await organizationApi.updateMemberRole(currentOrg.slug, userId, newRole);
      await loadMembers();
    } catch (err) {
      console.error("Failed to update role:", err);
      setError(t("settings.members.failedToUpdate"));
    }
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case "owner": return "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400";
      case "admin": return "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400";
      default: return "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300";
    }
  };

  const formatExpiryDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">{t("settings.members.title")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("settings.members.description")}
          </p>
        </div>
        <Button onClick={() => setShowInviteDialog(true)}>{t("settings.members.inviteMember")}</Button>
      </div>

      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg mb-4">
          <span>{error}</span>
          {errorType === "no_seats" && currentOrg && (
            <button
              onClick={() => router.push(`/${currentOrg.slug}/settings?scope=organization&tab=billing`)}
              className="ml-2 underline text-sm font-medium"
            >
              {t("settings.members.manageSeats")}
            </button>
          )}
          {errorType === "subscription_frozen" && currentOrg && (
            <button
              onClick={() => router.push(`/${currentOrg.slug}/settings?scope=organization&tab=billing`)}
              className="ml-2 underline text-sm font-medium"
            >
              {t("settings.members.renewSubscription")}
            </button>
          )}
          <button onClick={() => { setError(null); setErrorType("generic"); }} className="ml-4 underline text-sm">
            {t("settings.members.dismiss")}
          </button>
        </div>
      )}

      {successMessage && (
        <div className="bg-green-50 border border-green-200 text-green-800 dark:bg-green-900/20 dark:border-green-800 dark:text-green-400 px-4 py-3 rounded-lg mb-4">
          {successMessage}
          <button onClick={() => setSuccessMessage(null)} className="ml-4 underline text-sm">
            {t("settings.members.dismiss")}
          </button>
        </div>
      )}

      {/* Members List */}
      {loading ? (
        <div className="text-center py-8 text-muted-foreground">{t("settings.members.loading")}</div>
      ) : members.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {t("settings.members.noMembers")}
        </div>
      ) : (
        <div className="space-y-3">
          {members.map((member) => (
            <div
              key={member.id}
              className="flex items-center justify-between p-4 border border-border rounded-lg"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center text-sm font-medium">
                  {member.user?.name?.[0] || member.user?.username?.[0] || "?"}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">
                      {member.user?.name || member.user?.username || "Unknown"}
                    </span>
                    <span className={`text-xs px-2 py-0.5 rounded-full ${getRoleBadgeColor(member.role)}`}>
                      {member.role}
                    </span>
                    {member.user_id === user?.id && (
                      <span className="text-xs text-muted-foreground">{t("settings.members.you")}</span>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">{member.user?.email}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {member.role !== "owner" && member.user_id !== user?.id && (
                  <>
                    <select
                      value={member.role}
                      onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                      className="text-sm border border-border rounded px-2 py-1 bg-background"
                    >
                      <option value="member">{t("settings.members.roleMember")}</option>
                      <option value="admin">{t("settings.members.roleAdmin")}</option>
                    </select>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleRemove(member.user_id)}
                    >
                      {t("settings.members.remove")}
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pending Invitations */}
      {!loadingInvitations && pendingInvitations.length > 0 && (
        <div className="mt-6">
          <h3 className="text-sm font-semibold text-muted-foreground mb-3">
            {t("settings.members.pendingInvitations")}
          </h3>
          <div className="space-y-3">
            {pendingInvitations.map((invitation) => (
              <div
                key={invitation.id}
                className="flex items-center justify-between p-4 border border-dashed border-border rounded-lg"
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-muted/50 flex items-center justify-center text-sm font-medium text-muted-foreground">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <rect width="20" height="16" x="2" y="4" rx="2" />
                      <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7" />
                    </svg>
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{invitation.email}</span>
                      <span className={`text-xs px-2 py-0.5 rounded-full ${getRoleBadgeColor(invitation.role)}`}>
                        {invitation.role}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {t("settings.members.pendingExpires", { date: formatExpiryDate(invitation.expires_at) })}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleResend(invitation.id)}
                    disabled={resendingId === invitation.id}
                  >
                    {resendingId === invitation.id
                      ? t("settings.members.resending")
                      : t("settings.members.resend")}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => handleRevoke(invitation.id)}
                  >
                    {t("settings.members.revoke")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Invite Dialog */}
      {showInviteDialog && (
        <InviteDialog
          inviteEmail={inviteEmail}
          setInviteEmail={setInviteEmail}
          inviteRole={inviteRole}
          setInviteRole={setInviteRole}
          inviting={inviting}
          onInvite={handleInvite}
          onClose={() => {
            setShowInviteDialog(false);
            setInviteEmail("");
            setInviteRole("member");
            setError(null);
          }}
          t={t}
        />
      )}

      {/* Remove Member Confirmation Dialog */}
      <ConfirmDialog {...removeMemberDialog.dialogProps} />

      {/* Revoke Invitation Confirmation Dialog */}
      <ConfirmDialog {...revokeInvitationDialog.dialogProps} />
    </div>
  );
}

function InviteDialog({
  inviteEmail,
  setInviteEmail,
  inviteRole,
  setInviteRole,
  inviting,
  onInvite,
  onClose,
  t,
}: {
  inviteEmail: string;
  setInviteEmail: (email: string) => void;
  inviteRole: "admin" | "member";
  setInviteRole: (role: "admin" | "member") => void;
  inviting: boolean;
  onInvite: () => void;
  onClose: () => void;
  t: TranslationFn;
}) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">{t("settings.members.inviteDialog.title")}</h3>
        <div className="space-y-4">
          <FormField label={t("settings.members.inviteDialog.emailLabel")} htmlFor="invite-email">
            <Input
              id="invite-email"
              type="email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder={t("settings.members.inviteDialog.emailPlaceholder")}
            />
          </FormField>
          <FormField label={t("settings.members.inviteDialog.roleLabel")} htmlFor="invite-role">
            <select
              id="invite-role"
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value as "admin" | "member")}
              className="w-full border border-border rounded px-3 py-2 bg-background"
            >
              <option value="member">{t("settings.members.roleMember")}</option>
              <option value="admin">{t("settings.members.roleAdmin")}</option>
            </select>
          </FormField>
        </div>
        <div className="flex gap-3 mt-6">
          <Button variant="outline" className="flex-1" onClick={onClose}>
            {t("settings.members.inviteDialog.cancel")}
          </Button>
          <Button
            className="flex-1"
            onClick={onInvite}
            disabled={inviting || !inviteEmail}
          >
            {inviting ? t("settings.members.inviteDialog.inviting") : t("settings.members.inviteDialog.sendInvite")}
          </Button>
        </div>
      </div>
    </div>
  );
}
