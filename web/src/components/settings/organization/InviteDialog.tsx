import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
import type { TranslationFn } from "./GeneralSettings";

interface InviteDialogProps {
  inviteEmail: string;
  setInviteEmail: (email: string) => void;
  inviteRole: "admin" | "member";
  setInviteRole: (role: "admin" | "member") => void;
  inviting: boolean;
  onInvite: () => void;
  onClose: () => void;
  t: TranslationFn;
}

export function InviteDialog({
  inviteEmail,
  setInviteEmail,
  inviteRole,
  setInviteRole,
  inviting,
  onInvite,
  onClose,
  t,
}: InviteDialogProps) {
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
