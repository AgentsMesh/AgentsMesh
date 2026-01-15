"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { organizationApi } from "@/lib/api/client";

export type TranslationFn = (key: string, params?: Record<string, string | number>) => string;

interface GeneralSettingsProps {
  org: { name: string; slug: string } | null;
  t: TranslationFn;
}

export function GeneralSettings({ org, t }: GeneralSettingsProps) {
  const [name, setName] = useState(org?.name || "");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await organizationApi.update(org!.slug, { name });
    } catch (error) {
      console.error("Failed to save:", error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.organizationDetails.title")}</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.organizationDetails.nameLabel")}
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.organizationDetails.namePlaceholder")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.organizationDetails.slugLabel")}
            </label>
            <Input value={org?.slug || ""} disabled />
            <p className="text-xs text-muted-foreground mt-1">
              {t("settings.organizationDetails.slugHint")}
            </p>
          </div>
        </div>
        <div className="mt-6">
          <Button onClick={handleSave} disabled={saving}>
            {saving ? t("settings.organizationDetails.saving") : t("settings.organizationDetails.saveChanges")}
          </Button>
        </div>
      </div>

      <div className="border border-destructive rounded-lg p-6">
        <h2 className="text-lg font-semibold text-destructive mb-4">
          {t("settings.dangerZone.title")}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.dangerZone.description")}
        </p>
        <Button variant="destructive">{t("settings.dangerZone.deleteOrg")}</Button>
      </div>
    </div>
  );
}
