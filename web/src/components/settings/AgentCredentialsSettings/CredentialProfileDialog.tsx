"use client";

import { useState, useEffect, useCallback } from "react";
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
import type { CredentialProfileDialogProps, CredentialFormData } from "./types";

/**
 * CredentialProfileDialog - Dynamic dialog for adding or editing credential profiles.
 *
 * Renders form fields based on credentialFields (from AgentFile ENV SECRET/TEXT declarations).
 * Each field uses the full ENV name as key (e.g. "ANTHROPIC_API_KEY").
 */
export function CredentialProfileDialog({
  open,
  onOpenChange,
  credentialFields,
  editingProfile,
  onSubmit,
  t,
}: CredentialProfileDialogProps) {
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    if (editingProfile) {
      setFormName(editingProfile.name);
      setFormDescription(editingProfile.description || "");
      const values: Record<string, string> = {};
      for (const field of credentialFields) {
        if (field.type === "text" && editingProfile.configured_values?.[field.name]) {
          values[field.name] = editingProfile.configured_values[field.name];
        }
      }
      setFieldValues(values);
    } else {
      setFormName("");
      setFormDescription("");
      setFieldValues({});
    }
    setError(null);
  }, [open, editingProfile, credentialFields]);

  const handleFieldChange = useCallback((fieldName: string, value: string) => {
    setFieldValues((prev) => ({ ...prev, [fieldName]: value }));
  }, []);

  const handleSubmit = async () => {
    if (!formName.trim()) return;
    try {
      setSubmitting(true);
      setError(null);

      const credentials: Record<string, string> = {};
      for (const [key, value] of Object.entries(fieldValues)) {
        if (value) credentials[key] = value;
      }

      const formData: CredentialFormData = {
        name: formName,
        description: formDescription,
        credentials,
      };

      await onSubmit(formData);
      onOpenChange(false);
    } catch (err) {
      console.error("Failed to save profile:", err);
      setError(t("settings.agentCredentials.failedToSave"));
    } finally {
      setSubmitting(false);
    }
  };

  const getFieldLabel = (fieldName: string): string => {
    const i18nKey = `settings.agentCredentials.fields.${fieldName}`;
    const translated = t(i18nKey);
    return translated !== i18nKey ? translated : fieldName;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
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

        <div className="grid gap-4 py-4">
          {error && <div className="text-sm text-destructive">{error}</div>}

          <div className="grid gap-2">
            <Label htmlFor="cred-name">{t("settings.agentCredentials.name")}</Label>
            <Input
              id="cred-name"
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder={t("settings.agentCredentials.namePlaceholder")}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="cred-desc">{t("settings.agentCredentials.descriptionLabel")}</Label>
            <Textarea
              id="cred-desc"
              value={formDescription}
              onChange={(e) => setFormDescription(e.target.value)}
              placeholder={t("settings.agentCredentials.descriptionPlaceholder")}
              rows={2}
            />
          </div>

          {credentialFields.map((field) => (
            <div key={field.name} className="grid gap-2">
              <Label htmlFor={`cred-${field.name}`}>
                {getFieldLabel(field.name)}
                {field.optional && (
                  <span className="text-xs text-muted-foreground ml-1">
                    ({t("common.optional")})
                  </span>
                )}
              </Label>
              <Input
                id={`cred-${field.name}`}
                type={field.type === "secret" ? "password" : "text"}
                value={fieldValues[field.name] || ""}
                onChange={(e) => handleFieldChange(field.name, e.target.value)}
                placeholder={
                  editingProfile && field.type === "secret"
                    ? t("settings.agentCredentials.secretPlaceholder")
                    : ""
                }
              />
              {editingProfile && field.type === "secret" && (
                <p className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.secretEditHint")}
                </p>
              )}
            </div>
          ))}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={submitting || !formName.trim()}>
            {submitting
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

export default CredentialProfileDialog;
