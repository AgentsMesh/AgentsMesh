"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { SkillRegistry, SkillRegistryOverride } from "@/lib/api";
import { getExtensionService } from "@/lib/wasm-core";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import type { TranslationFn } from "../GeneralSettings";
import { PlatformRegistriesList } from "./PlatformRegistriesList";
import { OrgRegistriesList } from "./OrgRegistriesList";
import { AddRegistryDialog } from "./AddRegistryDialog";

interface SkillRegistriesSettingsProps {
  t: TranslationFn;
}

export function SkillRegistriesSettings({ t }: SkillRegistriesSettingsProps) {
  const [registries, setRegistries] = useState<SkillRegistry[]>([]);
  const [overrides, setOverrides] = useState<SkillRegistryOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [syncingId, setSyncingId] = useState<number | null>(null);
  const [togglingId, setTogglingId] = useState<number | null>(null);

  const loadRegistries = useCallback(async (signal?: AbortSignal) => {
    try {
      const [registriesRes, overridesRes] = await Promise.all([
        getExtensionService().list_skill_registries().then((j: string) => JSON.parse(j)),
        getExtensionService().list_skill_registry_overrides().then((j: string) => JSON.parse(j)),
      ]);
      if (signal?.aborted) return;
      setRegistries(registriesRes.skill_registries || []);
      setOverrides(overridesRes.overrides || []);
    } catch (error) {
      if (signal?.aborted) return;
      console.error("Failed to load skill registries:", error);
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    loadRegistries(controller.signal);
    return () => controller.abort();
  }, [loadRegistries]);

  const platformRegistries = useMemo(
    () => registries.filter((r) => r.organization_id == null),
    [registries]
  );
  const orgRegistries = useMemo(
    () => registries.filter((r) => r.organization_id != null),
    [registries]
  );

  const disabledRegistryIds = useMemo(() => {
    const ids = new Set<number>();
    for (const o of overrides) {
      if (o.is_disabled) ids.add(o.registry_id);
    }
    return ids;
  }, [overrides]);

  const handleTogglePlatformRegistry = useCallback(
    async (registryId: number, currentlyDisabled: boolean) => {
      setTogglingId(registryId);
      try {
        const res = JSON.parse(await getExtensionService().toggle_skill_registry(BigInt(registryId), JSON.stringify({ is_disabled: !currentlyDisabled })));
        setOverrides(res.overrides || []);
        toast.success(t("extensions.skillRegistries.toggleSuccess"));
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.skillRegistries.failedToToggle")));
      } finally {
        setTogglingId(null);
      }
    },
    [t]
  );

  const handleSync = useCallback(async (id: number) => {
    setSyncingId(id);
    try {
      await getExtensionService().sync_skill_registry(BigInt(id));
      toast.success(t("extensions.syncStarted"));
      loadRegistries();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToSync")));
    } finally {
      setSyncingId(null);
    }
  }, [t, loadRegistries]);

  const handleDelete = useCallback(async (id: number) => {
    if (!window.confirm(t("extensions.confirmDeleteSource"))) return;
    try {
      await getExtensionService().delete_skill_registry(BigInt(id));
      toast.success(t("extensions.sourceDeleted"));
      loadRegistries();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToDeleteSource")));
    }
  }, [t, loadRegistries]);

  const getSyncStatusVariant = (status: string): "default" | "secondary" | "destructive" | "outline" => {
    switch (status) {
      case "success": return "default";
      case "syncing": return "secondary";
      case "failed": return "destructive";
      default: return "outline";
    }
  };

  return (
    <div className="space-y-6">
      {/* Platform Registries Section */}
      <PlatformRegistriesList
        t={t}
        loading={loading}
        registries={platformRegistries}
        disabledRegistryIds={disabledRegistryIds}
        togglingId={togglingId}
        onToggle={handleTogglePlatformRegistry}
        getSyncStatusVariant={getSyncStatusVariant}
      />

      {/* Organization Registries Section */}
      <OrgRegistriesList
        t={t}
        loading={loading}
        registries={orgRegistries}
        syncingId={syncingId}
        onSync={handleSync}
        onDelete={handleDelete}
        onAdd={() => setShowAdd(true)}
        getSyncStatusVariant={getSyncStatusVariant}
      />

      {/* Add Registry Dialog */}
      <AddRegistryDialog
        t={t}
        open={showAdd}
        onOpenChange={setShowAdd}
        onAdded={loadRegistries}
      />
    </div>
  );
}
