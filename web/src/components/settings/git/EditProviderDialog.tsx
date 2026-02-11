"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
import { userRepositoryProviderApi, RepositoryProviderData } from "@/lib/api";
import { useTranslations } from "next-intl";

interface EditProviderDialogProps {
  provider: RepositoryProviderData;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * EditProviderDialog - Dialog for editing an existing Git provider
 */
export function EditProviderDialog({ provider, onClose, onSuccess }: EditProviderDialogProps) {
  const t = useTranslations();
  const [name, setName] = useState(provider.name);
  const [baseUrl, setBaseUrl] = useState(provider.base_url);
  const [botToken, setBotToken] = useState("");
  const [isActive, setIsActive] = useState(provider.is_active);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setSaving(true);
    setError(null);

    try {
      await userRepositoryProviderApi.update(provider.id, {
        name: name || undefined,
        base_url: baseUrl || undefined,
        bot_token: botToken || undefined,
        is_active: isActive,
      });
      onSuccess();
    } catch (err) {
      console.error("Failed to update provider:", err);
      setError(t("settings.gitSettings.providers.dialog.failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("settings.gitSettings.providers.dialog.editTitle")}</h2>
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

          <FormField
            label={t("settings.gitSettings.providers.dialog.name")}
            htmlFor="provider-name"
          >
            <Input
              id="provider-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>

          <FormField
            label={t("settings.gitSettings.providers.dialog.baseUrl")}
            htmlFor="provider-url"
          >
            <Input
              id="provider-url"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
            />
          </FormField>

          <FormField
            label={t("settings.gitSettings.providers.dialog.token")}
            htmlFor="provider-token"
            hint={t("settings.gitSettings.providers.dialog.tokenUpdateHint")}
          >
            <Input
              id="provider-token"
              type="password"
              value={botToken}
              onChange={(e) => setBotToken(e.target.value)}
            />
          </FormField>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">
              {t("settings.gitSettings.providers.dialog.active")}
            </label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
              />
              <div className="w-11 h-6 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-transparent after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
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
