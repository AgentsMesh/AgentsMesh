"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { useMeshStore, useTopology, type MeshNode } from "@/stores/mesh";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";
import { MeshFilterSection } from "./MeshFilterSection";

interface MeshSidebarContentProps {
  className?: string;
}

const STATUS_META: Record<string, { labelKey: string; color: string }> = {
  running: { labelKey: "ide.sidebar.mesh.statusRunning", color: "#3FB950" },
  initializing: { labelKey: "ide.sidebar.mesh.statusInit", color: "#D29922" },
  terminated: { labelKey: "ide.sidebar.mesh.statusTerminated", color: "#8B949E" },
  failed: { labelKey: "ide.sidebar.mesh.statusFailed", color: "#F85149" },
};

export function MeshSidebarContent({ className }: MeshSidebarContentProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const topology = useTopology();
  const fetchTopology = useMeshStore((s) => s.fetchTopology);

  const [searchQuery, setSearchQuery] = useState("");
  const [runnerFilter, setRunnerFilter] = useState<Set<string>>(new Set());
  const [statusFilter, setStatusFilter] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (currentOrg) fetchTopology();
  }, [currentOrg, fetchTopology]);

  const toggleSet = useCallback(
    (setter: React.Dispatch<React.SetStateAction<Set<string>>>, id: string) => {
      setter((prev) => {
        const next = new Set(prev);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
      });
    },
    [],
  );

  const runnerOptions = useMemo(() => {
    if (!topology?.nodes) return [];
    const counts = new Map<string, number>();
    for (const node of topology.nodes) {
      const rid = String((node as unknown as { runner_id?: string | number }).runner_id ?? "unknown");
      counts.set(rid, (counts.get(rid) ?? 0) + 1);
    }
    return Array.from(counts.entries()).map(([id, count]) => ({
      id,
      label: `${t("ide.sidebar.mesh.runnerPrefix")} ${id.slice(0, 8)}`,
      count,
      dotColor: "#3FB950",
    }));
  }, [topology?.nodes, t]);

  const statusOptions = useMemo(() => {
    if (!topology?.nodes) return [];
    const counts = new Map<string, number>();
    for (const node of topology.nodes) counts.set(node.status, (counts.get(node.status) ?? 0) + 1);
    return Array.from(counts.entries()).map(([id, count]) => ({
      id,
      label: t(STATUS_META[id]?.labelKey ?? "common.unknown"),
      count,
      dotColor: STATUS_META[id]?.color ?? "#6E7681",
    }));
  }, [topology?.nodes, t]);

  const filteredCount = useMemo(() => {
    return (topology?.nodes || []).filter((node: MeshNode) => {
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        if (!node.pod_key.toLowerCase().includes(q) && !node.model?.toLowerCase().includes(q)) return false;
      }
      if (runnerFilter.size > 0) {
        const rid = String((node as unknown as { runner_id?: string | number }).runner_id ?? "unknown");
        if (!runnerFilter.has(rid)) return false;
      }
      if (statusFilter.size > 0 && !statusFilter.has(node.status)) return false;
      return true;
    }).length;
  }, [topology?.nodes, searchQuery, runnerFilter, statusFilter]);

  const hasFilters = searchQuery !== "" || runnerFilter.size > 0 || statusFilter.size > 0;
  const handleReset = useCallback(() => {
    setSearchQuery("");
    setRunnerFilter(new Set());
    setStatusFilter(new Set());
  }, []);

  return (
    <div className={cn("flex h-full flex-col", className)}>
      <div className="px-3 py-3">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("ide.sidebar.mesh.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8 pl-8 text-[13px]"
          />
        </div>
      </div>

      <MeshFilterSection
        title={t("ide.sidebar.mesh.runnerFilter")}
        options={runnerOptions}
        selected={runnerFilter}
        onToggle={(id) => toggleSet(setRunnerFilter, id)}
      />

      <MeshFilterSection
        title={t("ide.sidebar.mesh.statusFilter")}
        options={statusOptions}
        selected={statusFilter}
        onToggle={(id) => toggleSet(setStatusFilter, id)}
      />

      <div className="mt-auto border-t border-border p-3">
        <button
          type="button"
          onClick={handleReset}
          disabled={!hasFilters}
          className={cn(
            "flex h-7 w-full items-center justify-center rounded-md border border-border bg-background text-xs font-medium text-foreground transition-colors",
            hasFilters ? "hover:bg-muted" : "cursor-not-allowed opacity-60",
          )}
        >
          {t("ide.sidebar.mesh.resetFilters")}
        </button>
        {hasFilters && (
          <div className="mt-2 text-center text-[11px] text-muted-foreground">
            {t("ide.sidebar.mesh.matchCount", { count: filteredCount })}
          </div>
        )}
      </div>
    </div>
  );
}

export default MeshSidebarContent;
