"use client";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { SkillRegistry } from "@/lib/api";
import { RefreshCw, Trash2, Plus, Lock, Globe } from "lucide-react";
import type { TranslationFn } from "../GeneralSettings";

interface OrgRegistriesListProps {
  t: TranslationFn;
  loading: boolean;
  registries: SkillRegistry[];
  syncingId: number | null;
  onSync: (id: number) => void;
  onDelete: (id: number) => void;
  onAdd: () => void;
  getSyncStatusVariant: (status: string) => "default" | "secondary" | "destructive" | "outline";
}

export function OrgRegistriesList({
  t,
  loading,
  registries,
  syncingId,
  onSync,
  onDelete,
  onAdd,
  getSyncStatusVariant,
}: OrgRegistriesListProps) {
  return (
    <div className="border border-border rounded-lg p-6">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">{t("extensions.skillRegistries.orgSources")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("extensions.skillRegistries.description")}
          </p>
        </div>
        <Button onClick={onAdd}>
          <Plus className="w-4 h-4 mr-1" />
          {t("extensions.skillRegistries.addSource")}
        </Button>
      </div>

      {loading ? (
        <div className="text-center py-4 text-muted-foreground">
          {t("extensions.loading")}
        </div>
      ) : registries.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {t("extensions.skillRegistries.noSources")}
        </div>
      ) : (
        <div className="space-y-3">
          {registries.map((registry) => (
            <div
              key={registry.id}
              className="border border-border rounded-lg p-4 flex items-center justify-between gap-4"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-medium truncate">{registry.repository_url}</span>
                  <Badge variant={getSyncStatusVariant(registry.sync_status)} className="text-xs shrink-0">
                    {registry.sync_status}
                  </Badge>
                  {registry.source_type && (
                    <Badge variant="outline" className="text-xs shrink-0">
                      {registry.source_type}
                    </Badge>
                  )}
                  {registry.auth_type && registry.auth_type !== "none" ? (
                    <Badge variant="secondary" className="text-xs shrink-0">
                      <Lock className="w-3 h-3 mr-1" />
                      {registry.auth_type.replace("_", " ").toUpperCase()}
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-xs shrink-0">
                      <Globe className="w-3 h-3 mr-1" />
                      {t("extensions.skillRegistries.public")}
                    </Badge>
                  )}
                </div>
                <div className="flex items-center gap-4 text-xs text-muted-foreground">
                  <span>{t("extensions.skillRegistries.skillCount")}: {registry.skill_count}</span>
                  {registry.branch && <span>{t("extensions.branch")}: {registry.branch}</span>}
                  {registry.last_synced_at && (
                    <span>{t("extensions.skillRegistries.lastSynced")}: {new Date(registry.last_synced_at).toLocaleString()}</span>
                  )}
                </div>
                {registry.compatible_agents && registry.compatible_agents.length > 0 && (
                  <div className="flex items-center gap-1 mt-1">
                    <span className="text-xs text-muted-foreground">{t("extensions.skillRegistries.compatibleAgents")}:</span>
                    {registry.compatible_agents.map((agent) => (
                      <Badge key={agent} variant="outline" className="text-xs">
                        {agent}
                      </Badge>
                    ))}
                  </div>
                )}
                {registry.sync_error && (
                  <p className="text-xs text-destructive mt-1">{registry.sync_error}</p>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={syncingId === registry.id}
                  onClick={() => onSync(registry.id)}
                >
                  <RefreshCw className={`w-4 h-4 ${syncingId === registry.id ? "animate-spin" : ""}`} />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onDelete(registry.id)}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
