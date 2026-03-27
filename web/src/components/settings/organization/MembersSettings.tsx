"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useAuthStore } from "@/stores/auth";
import { organizationApi } from "@/lib/api";
import { ApiError } from "@/lib/api/base";
import { isApiErrorCode } from "@/lib/api/errors";
import { invitationApi, type Invitation } from "@/lib/api/invitation";
import type { TranslationFn } from "./GeneralSettings";
import { MembersList, type Member } from "./MembersList";
import { PendingInvitations } from "./PendingInvitations";
import { InviteDialog } from "./InviteDialog";

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

  useEffect(() => { loadMembers(); loadInvitations(); }, [loadMembers, loadInvitations]);

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
      if (isApiErrorCode(err, "NO_AVAILABLE_SEATS")) {
        setError(t("settings.members.noSeats")); setErrorType("no_seats");
      } else if (isApiErrorCode(err, "SUBSCRIPTION_FROZEN")) {
        setError(t("settings.members.subscriptionFrozen")); setErrorType("subscription_frozen");
      } else if (isApiErrorCode(err, "ALREADY_EXISTS")) {
        const msg = (err as ApiError).serverMessage || "";
        setError(msg.includes("pending invitation") ? t("settings.members.pendingExists") : t("settings.members.alreadyMember"));
      } else {
        setError(t("settings.members.failedToInvite"));
      }
    } finally { setInviting(false); }
  };

  const handleRevoke = async (invitationId: number) => {
    const confirmed = await revokeInvitationDialog.confirm();
    if (!confirmed) return;
    try { await invitationApi.revoke(invitationId); setSuccessMessage(t("settings.members.invitationRevoked")); await loadInvitations(); }
    catch (err) { console.error("Failed to revoke invitation:", err); setError(t("settings.members.failedToRevoke")); }
  };

  const handleResend = async (invitationId: number) => {
    setResendingId(invitationId);
    try { await invitationApi.resend(invitationId); setSuccessMessage(t("settings.members.invitationResent")); }
    catch (err) { console.error("Failed to resend invitation:", err); setError(t("settings.members.failedToResend")); }
    finally { setResendingId(null); }
  };

  const handleRemove = async (userId: number) => {
    if (!currentOrg) return;
    const confirmed = await removeMemberDialog.confirm();
    if (!confirmed) return;
    try { await organizationApi.removeMember(currentOrg.slug, userId); await loadMembers(); }
    catch (err) { console.error("Failed to remove member:", err); setError(t("settings.members.failedToRemove")); }
  };

  const handleRoleChange = async (userId: number, newRole: string) => {
    if (!currentOrg) return;
    try { await organizationApi.updateMemberRole(currentOrg.slug, userId, newRole); await loadMembers(); }
    catch (err) { console.error("Failed to update role:", err); setError(t("settings.members.failedToUpdate")); }
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">{t("settings.members.title")}</h2>
          <p className="text-sm text-muted-foreground">{t("settings.members.description")}</p>
        </div>
        <Button onClick={() => setShowInviteDialog(true)}>{t("settings.members.inviteMember")}</Button>
      </div>

      {error && (
        <ErrorBanner error={error} errorType={errorType} orgSlug={currentOrg?.slug} t={t}
          onDismiss={() => { setError(null); setErrorType("generic"); }}
          onNavigate={(path) => router.push(path)} />
      )}

      {successMessage && (
        <div className="bg-green-50 border border-green-200 text-green-800 dark:bg-green-900/20 dark:border-green-800 dark:text-green-400 px-4 py-3 rounded-lg mb-4">
          {successMessage}
          <button onClick={() => setSuccessMessage(null)} className="ml-4 underline text-sm">{t("settings.members.dismiss")}</button>
        </div>
      )}

      <MembersList members={members} loading={loading} currentUserId={user?.id} t={t}
        onRoleChange={handleRoleChange} onRemove={handleRemove} />

      <PendingInvitations invitations={pendingInvitations} loading={loadingInvitations}
        resendingId={resendingId} t={t} onResend={handleResend} onRevoke={handleRevoke} />

      {showInviteDialog && (
        <InviteDialog inviteEmail={inviteEmail} setInviteEmail={setInviteEmail}
          inviteRole={inviteRole} setInviteRole={setInviteRole} inviting={inviting}
          onInvite={handleInvite}
          onClose={() => { setShowInviteDialog(false); setInviteEmail(""); setInviteRole("member"); setError(null); }}
          t={t} />
      )}

      <ConfirmDialog {...removeMemberDialog.dialogProps} />
      <ConfirmDialog {...revokeInvitationDialog.dialogProps} />
    </div>
  );
}

function ErrorBanner({ error, errorType, orgSlug, t, onDismiss, onNavigate }: {
  error: string; errorType: string; orgSlug?: string; t: TranslationFn;
  onDismiss: () => void; onNavigate: (path: string) => void;
}) {
  return (
    <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg mb-4">
      <span>{error}</span>
      {errorType === "no_seats" && orgSlug && (
        <button onClick={() => onNavigate(`/${orgSlug}/settings?scope=organization&tab=billing`)}
          className="ml-2 underline text-sm font-medium">{t("settings.members.manageSeats")}</button>
      )}
      {errorType === "subscription_frozen" && orgSlug && (
        <button onClick={() => onNavigate(`/${orgSlug}/settings?scope=organization&tab=billing`)}
          className="ml-2 underline text-sm font-medium">{t("settings.members.renewSubscription")}</button>
      )}
      <button onClick={onDismiss} className="ml-4 underline text-sm">{t("settings.members.dismiss")}</button>
    </div>
  );
}
