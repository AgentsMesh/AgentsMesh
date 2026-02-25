"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { organizationApi } from "@/lib/api";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { useAuthStore } from "@/stores/auth";
import { toast } from "sonner";

export type TranslationFn = (key: string, params?: Record<string, string | number>) => string;

interface GeneralSettingsProps {
  org: { name: string; slug: string } | null;
  t: TranslationFn;
}

export function GeneralSettings({ org, t }: GeneralSettingsProps) {
  const [name, setName] = useState(org?.name || "");
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const router = useRouter();
  const { currentOrg, organizations, setCurrentOrg, setOrganizations } = useAuthStore();

  const deleteOrgDialog = useConfirmDialog({
    title: t("settings.dangerZone.title"),
    description: t("settings.dangerZone.description"),
    confirmText: t("settings.dangerZone.deleteOrg"),
    variant: "destructive",
  });

  const handleSave = async () => {
    setSaving(true);
    try {
      await organizationApi.update(org!.slug, { name });

      // Sync Zustand store cache
      if (currentOrg && currentOrg.slug === org!.slug) {
        setCurrentOrg({ ...currentOrg, name });
      }
      setOrganizations(
        organizations.map((o) =>
          o.slug === org!.slug ? { ...o, name } : o
        )
      );

      toast.success(t("settings.organizationDetails.saveSuccess") || "Saved");
    } catch (error) {
      console.error("Failed to save:", error);
      toast.error(getLocalizedErrorMessage(error, t, t("settings.organizationDetails.saveFailed") || "Failed to save"));
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteOrg = async () => {
    const confirmed = await deleteOrgDialog.confirm();
    if (!confirmed) return;

    setDeleting(true);
    try {
      await organizationApi.delete(org!.slug);

      // Remove from store and redirect
      const remaining = organizations.filter((o) => o.slug !== org!.slug);
      setOrganizations(remaining);
      if (remaining.length > 0) {
        setCurrentOrg(remaining[0]);
        router.push(`/${remaining[0].slug}/workspace`);
      } else {
        router.push("/");
      }

      toast.success(t("settings.dangerZone.deleteSuccess") || "Organization deleted");
    } catch (error) {
      console.error("Failed to delete organization:", error);
      toast.error(getLocalizedErrorMessage(error, t, t("settings.dangerZone.deleteFailed") || "Failed to delete"));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.organizationDetails.title")}</h2>
        <div className="space-y-4">
          <FormField
            label={t("settings.organizationDetails.nameLabel")}
            htmlFor="org-name"
          >
            <Input
              id="org-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.organizationDetails.namePlaceholder")}
            />
          </FormField>

          <FormField
            label={t("settings.organizationDetails.slugLabel")}
            htmlFor="org-slug"
            hint={t("settings.organizationDetails.slugHint")}
          >
            <Input id="org-slug" value={org?.slug || ""} disabled />
          </FormField>
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
        <Button variant="destructive" onClick={handleDeleteOrg} disabled={deleting}>
          {deleting ? t("settings.dangerZone.deleting") || "Deleting..." : t("settings.dangerZone.deleteOrg")}
        </Button>
      </div>

      <ConfirmDialog {...deleteOrgDialog.dialogProps} />
    </div>
  );
}
