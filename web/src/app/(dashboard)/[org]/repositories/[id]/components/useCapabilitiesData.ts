"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { extensionApi, InstalledSkill, InstalledMcpServer } from "@/lib/api";
import { useConfirmDialog } from "@/components/ui/confirm-dialog";

export function useCapabilitiesData(repositoryId: number) {
  const t = useTranslations();
  const [orgSkills, setOrgSkills] = useState<InstalledSkill[]>([]);
  const [userSkills, setUserSkills] = useState<InstalledSkill[]>([]);
  const [orgMcpServers, setOrgMcpServers] = useState<InstalledMcpServer[]>([]);
  const [userMcpServers, setUserMcpServers] = useState<InstalledMcpServer[]>([]);
  const [loading, setLoading] = useState(true);

  const { dialogProps: confirmDialogProps, confirm } = useConfirmDialog();

  const loadSkills = useCallback(async (mounted?: { current: boolean }) => {
    try {
      const [orgRes, userRes] = await Promise.all([
        extensionApi.listRepoSkills(repositoryId, "org"),
        extensionApi.listRepoSkills(repositoryId, "user"),
      ]);
      if (mounted && !mounted.current) return;
      setOrgSkills(orgRes.skills || []);
      setUserSkills(userRes.skills || []);
    } catch (error) {
      if (mounted && !mounted.current) return;
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToLoadSkills")));
    }
  }, [repositoryId, t]);

  const loadMcpServers = useCallback(async (mounted?: { current: boolean }) => {
    try {
      const [orgRes, userRes] = await Promise.all([
        extensionApi.listRepoMcpServers(repositoryId, "org"),
        extensionApi.listRepoMcpServers(repositoryId, "user"),
      ]);
      if (mounted && !mounted.current) return;
      setOrgMcpServers(orgRes.mcp_servers || []);
      setUserMcpServers(userRes.mcp_servers || []);
    } catch (error) {
      if (mounted && !mounted.current) return;
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToLoadMcpServers")));
    }
  }, [repositoryId, t]);

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
    try { await extensionApi.updateSkill(repositoryId, skill.id, { is_enabled: !skill.is_enabled }); await loadSkills(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate"))); }
  }, [repositoryId, loadSkills, t]);

  const handleDeleteSkill = useCallback(async (skill: InstalledSkill) => {
    const confirmed = await confirm({
      title: t("extensions.confirmUninstallSkill"),
      description: t("extensions.uninstallSkillDescription", { name: skill.slug }),
      variant: "destructive", confirmText: t("extensions.uninstall"), cancelText: t("extensions.cancel"),
    });
    if (!confirmed) return;
    try { await extensionApi.uninstallSkill(repositoryId, skill.id); toast.success(t("extensions.uninstalled")); await loadSkills(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUninstall"))); }
  }, [repositoryId, loadSkills, t, confirm]);

  const handleToggleMcp = useCallback(async (mcp: InstalledMcpServer) => {
    try { await extensionApi.updateMcpServer(repositoryId, mcp.id, { is_enabled: !mcp.is_enabled }); await loadMcpServers(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate"))); }
  }, [repositoryId, loadMcpServers, t]);

  const handleDeleteMcp = useCallback(async (mcp: InstalledMcpServer) => {
    const confirmed = await confirm({
      title: t("extensions.confirmUninstallMcp"),
      description: t("extensions.uninstallMcpDescription", { name: mcp.name || mcp.slug }),
      variant: "destructive", confirmText: t("extensions.uninstall"), cancelText: t("extensions.cancel"),
    });
    if (!confirmed) return;
    try { await extensionApi.uninstallMcpServer(repositoryId, mcp.id); toast.success(t("extensions.uninstalled")); await loadMcpServers(); }
    catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUninstall"))); }
  }, [repositoryId, loadMcpServers, t, confirm]);

  return {
    orgSkills, userSkills, orgMcpServers, userMcpServers, loading,
    loadSkills, loadMcpServers,
    handleToggleSkill, handleDeleteSkill, handleToggleMcp, handleDeleteMcp,
    confirmDialogProps,
  };
}
