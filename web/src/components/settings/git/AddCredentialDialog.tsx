"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
import { userGitCredentialApi } from "@/lib/api";
import { useTranslations } from "next-intl";

interface AddCredentialDialogProps {
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * AddCredentialDialog - Dialog for adding a new Git credential (PAT or SSH Key)
 */
export function AddCredentialDialog({ onClose, onSuccess }: AddCredentialDialogProps) {
  const t = useTranslations();
  const [credentialType, setCredentialType] = useState<"pat" | "ssh_key">("pat");
  const [name, setName] = useState("");
  const [pat, setPat] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [hostPattern, setHostPattern] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!name) {
      setError(t("settings.gitSettings.credentials.dialog.nameRequired"));
      return;
    }

    if (credentialType === "pat" && !pat) {
      setError(t("settings.gitSettings.credentials.dialog.patRequired"));
      return;
    }

    if (credentialType === "ssh_key" && !privateKey) {
      setError(t("settings.gitSettings.credentials.dialog.sshRequired"));
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await userGitCredentialApi.create({
        name,
        credential_type: credentialType,
        pat: credentialType === "pat" ? pat : undefined,
        private_key: credentialType === "ssh_key" ? privateKey : undefined,
        host_pattern: hostPattern || undefined,
      });
      onSuccess();
    } catch (err) {
      console.error("Failed to create credential:", err);
      setError(t("settings.gitSettings.credentials.dialog.failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("settings.gitSettings.credentials.dialog.title")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        <div className="p-4 space-y-4">
          {error && (
            <div className="p-3 bg-destructive/10 text-destructive text-sm rounded-lg">
              {error}
            </div>
          )}

          <FormField label={t("settings.gitSettings.credentials.dialog.type")}>
            <div className="flex gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  checked={credentialType === "pat"}
                  onChange={() => setCredentialType("pat")}
                  className="w-4 h-4"
                />
                <span className="text-sm">Personal Access Token</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  checked={credentialType === "ssh_key"}
                  onChange={() => setCredentialType("ssh_key")}
                  className="w-4 h-4"
                />
                <span className="text-sm">SSH Key</span>
              </label>
            </div>
          </FormField>

          <FormField
            label={t("settings.gitSettings.credentials.dialog.name")}
            htmlFor="credential-name"
          >
            <Input
              id="credential-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.gitSettings.credentials.dialog.namePlaceholder")}
            />
          </FormField>

          {credentialType === "pat" && (
            <FormField
              label="Personal Access Token"
              htmlFor="credential-pat"
              hint={t("settings.gitSettings.credentials.dialog.patHint")}
            >
              <Input
                id="credential-pat"
                type="password"
                value={pat}
                onChange={(e) => setPat(e.target.value)}
                placeholder="ghp_xxx or glpat-xxx"
              />
            </FormField>
          )}

          {credentialType === "ssh_key" && (
            <FormField
              label={t("settings.gitSettings.credentials.dialog.privateKey")}
              htmlFor="credential-ssh"
            >
              <textarea
                id="credential-ssh"
                value={privateKey}
                onChange={(e) => setPrivateKey(e.target.value)}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                className="flex min-h-[120px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              />
            </FormField>
          )}

          <FormField
            label={t("settings.gitSettings.credentials.dialog.hostPattern")}
            htmlFor="credential-host"
            hint={t("settings.gitSettings.credentials.dialog.hostPatternHint")}
          >
            <Input
              id="credential-host"
              value={hostPattern}
              onChange={(e) => setHostPattern(e.target.value)}
              placeholder="github.com, gitlab.company.com"
            />
          </FormField>
        </div>

        <div className="flex justify-end gap-3 p-4 border-t border-border">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={saving}>
            {saving ? t("common.loading") : t("common.save")}
          </Button>
        </div>
      </div>
    </div>
  );
}
