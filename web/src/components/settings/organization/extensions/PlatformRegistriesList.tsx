"use client";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { SkillRegistry } from "@/lib/api";
import type { TranslationFn } from "../GeneralSettings";

interface PlatformRegistriesListProps {
  t: TranslationFn;
  loading: boolean;
  registries: SkillRegistry[];
  disabledRegistryIds: Set<number>;
  togglingId: number | null;
  onToggle: (registryId: number, currentlyDisabled: boolean) => void;
  getSyncStatusVariant: (status: string) => "default" | "secondary" | "destructive" | "outline";
}

export function PlatformRegistriesList({
  t,
  loading,
  registries,
  disabledRegistryIds,
  togglingId,
  onToggle,
  getSyncStatusVariant,
}: PlatformRegistriesListProps) {
  return (
    <div className="border border-border rounded-lg p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold">{t("extensions.skillRegistries.platformSources")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("extensions.skillRegistries.platformSourcesDescription")}
        </p>
      </div>

      {loading ? (
        <div className="text-center py-4 text-muted-foreground">
          {t("extensions.loading")}
        </div>
      ) : registries.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {t("extensions.skillRegistries.noPlatformSources")}
        </div>
      ) : (
        <div className="space-y-3">
          {registries.map((registry) => {
            const isDisabled = disabledRegistryIds.has(registry.id);
            return (
              <div
                key={registry.id}
                className={`border border-border rounded-lg p-4 flex items-center justify-between gap-4 ${
                  isDisabled ? "opacity-60" : ""
                }`}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium truncate">{registry.repository_url}</span>
                    <Badge variant={getSyncStatusVariant(registry.sync_status)} className="text-xs shrink-0">
                      {registry.sync_status}
                    </Badge>
                    <Badge variant="secondary" className="text-xs shrink-0">
                      {t("extensions.skillRegistries.platform")}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>{t("extensions.skillRegistries.skillCount")}: {registry.skill_count}</span>
                    {registry.branch && <span>{t("extensions.branch")}: {registry.branch}</span>}
                    {registry.last_synced_at && (
                      <span>{t("extensions.skillRegistries.lastSynced")}: {new Date(registry.last_synced_at).toLocaleString()}</span>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Button
                    variant={isDisabled ? "outline" : "default"}
                    size="sm"
                    disabled={togglingId === registry.id}
                    onClick={() => onToggle(registry.id, isDisabled)}
                  >
                    {isDisabled
                      ? t("extensions.skillRegistries.disabled")
                      : t("extensions.skillRegistries.enabled")}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
