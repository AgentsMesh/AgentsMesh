"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
import { userRepositoryProviderApi } from "@/lib/api";
import { useTranslations } from "next-intl";
import { ChevronLeft } from "lucide-react";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";

interface AddProviderDialogProps {
  onClose: () => void;
  onSuccess: () => void;
}

const PROVIDERS = [
  { type: "github", name: "GitHub", defaultUrl: "https://github.com" },
  { type: "gitlab", name: "GitLab", defaultUrl: "https://gitlab.com" },
  { type: "gitee", name: "Gitee", defaultUrl: "https://gitee.com" },
];

/**
 * AddProviderDialog - Dialog for adding a new Git provider
 */
export function AddProviderDialog({ onClose, onSuccess }: AddProviderDialogProps) {
  const t = useTranslations();
  const [step, setStep] = useState<"type" | "details">("type");
  const [providerType, setProviderType] = useState("");
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [botToken, setBotToken] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectType = (type: string) => {
    const provider = PROVIDERS.find(p => p.type === type);
    setProviderType(type);
    setName(provider?.name || "");
    setBaseUrl(provider?.defaultUrl || "");
    setStep("details");
  };

  const handleSubmit = async () => {
    if (!name || !baseUrl || !botToken) {
      setError(t("settings.gitSettings.providers.dialog.fillAll"));
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await userRepositoryProviderApi.create({
        provider_type: providerType,
        name,
        base_url: baseUrl,
        bot_token: botToken,
      });
      onSuccess();
    } catch (err) {
      console.error("Failed to create provider:", err);
      setError(t("settings.gitSettings.providers.dialog.failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("settings.gitSettings.providers.dialog.title")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        <div className="p-4">
          {error && (
            <div className="mb-4 p-3 bg-destructive/10 text-destructive text-sm rounded-lg">
              {error}
            </div>
          )}

          {step === "type" && (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground mb-4">
                {t("settings.gitSettings.providers.dialog.selectType")}
              </p>
              {PROVIDERS.map((provider) => (
                <button
                  key={provider.type}
                  onClick={() => selectType(provider.type)}
                  className="w-full flex items-center gap-4 p-4 border border-border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <GitProviderIcon provider={provider.type} />
                  <span className="font-medium">{provider.name}</span>
                </button>
              ))}
            </div>
          )}

          {step === "details" && (
            <div className="space-y-4">
              <button
                onClick={() => setStep("type")}
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              >
                <ChevronLeft className="w-4 h-4" />
                {t("common.back")}
              </button>

              <FormField
                label={t("settings.gitSettings.providers.dialog.name")}
                htmlFor="provider-name"
              >
                <Input
                  id="provider-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="My GitHub"
                />
              </FormField>

              <FormField
                label={t("settings.gitSettings.providers.dialog.baseUrl")}
                htmlFor="provider-url"
                hint={t("settings.gitSettings.providers.dialog.baseUrlHint")}
              >
                <Input
                  id="provider-url"
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="https://github.com"
                />
              </FormField>

              <FormField
                label={t("settings.gitSettings.providers.dialog.token")}
                htmlFor="provider-token"
                hint={t("settings.gitSettings.providers.dialog.tokenHint")}
              >
                <Input
                  id="provider-token"
                  type="password"
                  value={botToken}
                  onChange={(e) => setBotToken(e.target.value)}
                  placeholder="ghp_xxx or glpat-xxx"
                />
              </FormField>
            </div>
          )}
        </div>

        {step === "details" && (
          <div className="flex justify-end gap-3 p-4 border-t border-border">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleSubmit} disabled={saving}>
              {saving ? t("common.loading") : t("common.save")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
