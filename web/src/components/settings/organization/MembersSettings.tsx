"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/stores/auth";
import { organizationApi } from "@/lib/api/client";
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
  const { currentOrg, user } = useAuthStore();
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [inviting, setInviting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!currentOrg) return;
    try {
      setLoading(true);
      const response = await organizationApi.listMembers(currentOrg.slug);
      setMembers(response.members || []);
    } catch (err) {
      console.error("Failed to load members:", err);
      setError("Failed to load members");
    } finally {
      setLoading(false);
    }
  }, [currentOrg]);

  useEffect(() => {
    loadMembers();
  }, [loadMembers]);

  const handleInvite = async () => {
    if (!currentOrg || !inviteEmail) return;
    setInviting(true);
    setError(null);
    try {
      await organizationApi.inviteMember(currentOrg.slug, inviteEmail, inviteRole);
      setShowInviteDialog(false);
      setInviteEmail("");
      setInviteRole("member");
      await loadMembers();
    } catch (err) {
      console.error("Failed to invite member:", err);
      setError("Failed to invite member. Please check the email and try again.");
    } finally {
      setInviting(false);
    }
  };

  const handleRemove = async (userId: number) => {
    if (!currentOrg) return;
    if (!confirm("Are you sure you want to remove this member?")) return;
    try {
      await organizationApi.removeMember(currentOrg.slug, userId);
      await loadMembers();
    } catch (err) {
      console.error("Failed to remove member:", err);
      setError("Failed to remove member");
    }
  };

  const handleRoleChange = async (userId: number, newRole: string) => {
    if (!currentOrg) return;
    try {
      await organizationApi.updateMemberRole(currentOrg.slug, userId, newRole);
      await loadMembers();
    } catch (err) {
      console.error("Failed to update role:", err);
      setError("Failed to update member role");
    }
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case "owner": return "bg-purple-100 text-purple-800";
      case "admin": return "bg-blue-100 text-blue-800";
      default: return "bg-gray-100 text-gray-800";
    }
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
          {error}
          <button onClick={() => setError(null)} className="ml-4 underline text-sm">
            {t("settings.members.dismiss")}
          </button>
        </div>
      )}

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
          }}
          t={t}
        />
      )}
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
  inviteRole: string;
  setInviteRole: (role: string) => void;
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
          <div>
            <label className="block text-sm font-medium mb-2">{t("settings.members.inviteDialog.emailLabel")}</label>
            <Input
              type="email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder={t("settings.members.inviteDialog.emailPlaceholder")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t("settings.members.inviteDialog.roleLabel")}</label>
            <select
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value)}
              className="w-full border border-border rounded px-3 py-2 bg-background"
            >
              <option value="member">{t("settings.members.roleMember")}</option>
              <option value="admin">{t("settings.members.roleAdmin")}</option>
            </select>
          </div>
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
