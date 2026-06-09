"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CredentialFormFields } from "./CredentialFormFields";
import {
  useCredentialDialogForm,
  type CredentialFormDialogProps,
} from "./useCredentialDialogForm";

export function CredentialFormDialog(props: CredentialFormDialogProps) {
  const { open, onOpenChange, editingProfile, t } = props;
  const form = useCredentialDialogForm(props);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {editingProfile
              ? t("settings.agentCredentials.editProfile")
              : t("settings.agentCredentials.addProfile")}
          </DialogTitle>
          <DialogDescription>
            {t("settings.agentCredentials.customProfileDescription")}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 px-6 py-4">
          {form.error && <div className="text-sm text-destructive">{form.error}</div>}

          <div className="grid gap-2">
            <Label htmlFor="cred-name">{t("settings.agentCredentials.name")}</Label>
            <Input
              id="cred-name"
              value={form.formName}
              onChange={(e) => form.setFormName(e.target.value)}
              placeholder={t("settings.agentCredentials.namePlaceholder")}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="cred-desc">{t("settings.agentCredentials.descriptionLabel")}</Label>
            <Textarea
              id="cred-desc"
              value={form.formDescription}
              onChange={(e) => form.setFormDescription(e.target.value)}
              placeholder={t("settings.agentCredentials.descriptionPlaceholder")}
              rows={2}
            />
          </div>

          <CredentialFormFields
            spec={form.spec}
            values={form.formState.values}
            onValueChange={form.onValueChange}
            selectedOneOf={form.formState.selectedOneOf}
            onOneOfChange={form.onOneOfChange}
            customEnv={form.formState.customEnv}
            onCustomEnvChange={form.onCustomEnvChange}
            configuredKeys={form.previousKeys}
            removedKeys={form.formState.removedKeys}
            onRemoveKey={form.onRemoveKey}
            onRestoreKey={form.onRestoreKey}
            isEditing={!!editingProfile}
            t={t}
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={form.handleSubmit}
            disabled={form.submitting || !form.formName.trim() || form.customEnvInvalid}
          >
            {form.submitting
              ? t("common.saving")
              : editingProfile
                ? t("common.save")
                : t("common.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default CredentialFormDialog;
