"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import type { InstalledSkill, InstalledMcpServer } from "@/lib/api/extensionTypes";
import { listRepoSkills, updateSkill, uninstallSkill } from "@/lib/api/repoSkillExtension";
import { listRepoMcpServers, updateMcpServer, uninstallMcpServer } from "@/lib/api/repoMcpExtension";
import { useCurrentOrg } from "@/stores/auth";
import { useConfirmDialog } from "@/components/ui/confirm-dialog";

export function useCapabilitiesData(repositoryId: number) {
  const t = useTranslations();
  const orgSlug = useCurrentOrg()?.slug || "";
  const [orgSkills, setOrgSkills] = useState<InstalledSkill[]>([]);
  const [userSkills, setUserSkills] = useState<InstalledSkill[]>([]);
  const [orgMcpServers, setOrgMcpServers] = useState<InstalledMcpServer[]>([]);
  const [userMcpServers, setUserMcpServers] = useState<InstalledMcpServer[]>([]);
  const [loading, setLoading] = useState(true);

  const { dialogProps: confirmDialogProps, confirm } = useConfirmDialog();

  const loadSkills = useCallback(async (mounted?: { current: boolean }) => {
    if (!orgSlug) return;
    try {
      const [orgRes, userRes] = await Promise.all([
        listRepoSkills(orgSlug, repositoryId, { scope: "org" }),
        listRepoSkills(orgSlug, repositoryId, { scope: "user" }),
      ]);
      if (mounted && !mounted.current) return;
      setOrgSkills(orgRes.items || []);
      setUserSkills(userRes.items || []);
    } catch (error) {
      if (mounted && !mounted.current) return;
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToLoadSkills")));
    }
  }, [orgSlug, repositoryId, t]);

  const loadMcpServers = useCallback(async (mounted?: { current: boolean }) => {
    if (!orgSlug) return;
    try {
      const [orgRes, userRes] = await Promise.all([
        listRepoMcpServers(orgSlug, repositoryId, { scope: "org" }),
        listRepoMcpServers(orgSlug, repositoryId, { scope: "user" }),
      ]);
      if (mounted && !mounted.current) return;
      setOrgMcpServers(orgRes.items || []);
      setUserMcpServers(userRes.items || []);
    } catch (error) {
      if (mounted && !mounted.current) return;
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToLoadMcpServers")));
    }
  }, [orgSlug, repositoryId, t]);

  useEffect(() => {
    const mounted = { current: true };
    const load = async () => {
      setLoading(true);
      await Promise.all([loadSkills(mounted), loadMcpServers(mounted)]);
      if (mounted.current) setLoading(false);
    };
    load();
    return () => { mounted.current = false; };
  }, [loadSkills, loadMcpServers]);

  const handleToggleSkill = useCallback(async (skill: InstalledSkill) => {
    if (!orgSlug) return;
    try { await updateSkill(orgSlug, repositoryId, skill.id, { isEnabled: !skill.is_enabled }); await loadSkills(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate"))); }
  }, [orgSlug, repositoryId, loadSkills, t]);

  const handleDeleteSkill = useCallback(async (skill: InstalledSkill) => {
    if (!orgSlug) return;
    const confirmed = await confirm({
      title: t("extensions.confirmUninstallSkill"),
      description: t("extensions.uninstallSkillDescription", { name: skill.slug }),
      variant: "destructive", confirmText: t("extensions.uninstall"), cancelText: t("extensions.cancel"),
    });
    if (!confirmed) return;
    try { await uninstallSkill(orgSlug, repositoryId, skill.id); toast.success(t("extensions.uninstalled")); await loadSkills(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUninstall"))); }
  }, [orgSlug, repositoryId, loadSkills, t, confirm]);

  const handleToggleMcp = useCallback(async (mcp: InstalledMcpServer) => {
    if (!orgSlug) return;
    try { await updateMcpServer(orgSlug, repositoryId, mcp.id, { isEnabled: !mcp.is_enabled }); await loadMcpServers(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate"))); }
  }, [orgSlug, repositoryId, loadMcpServers, t]);

  const handleDeleteMcp = useCallback(async (mcp: InstalledMcpServer) => {
    if (!orgSlug) return;
    const confirmed = await confirm({
      title: t("extensions.confirmUninstallMcp"),
      description: t("extensions.uninstallMcpDescription", { name: mcp.name || mcp.slug }),
      variant: "destructive", confirmText: t("extensions.uninstall"), cancelText: t("extensions.cancel"),
    });
    if (!confirmed) return;
    try { await uninstallMcpServer(orgSlug, repositoryId, mcp.id); toast.success(t("extensions.uninstalled")); await loadMcpServers(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUninstall"))); }
  }, [orgSlug, repositoryId, loadMcpServers, t, confirm]);

  return {
    orgSkills, userSkills, orgMcpServers, userMcpServers, loading,
    loadSkills, loadMcpServers,
    handleToggleSkill, handleDeleteSkill, handleToggleMcp, handleDeleteMcp,
    confirmDialogProps,
  };
}
