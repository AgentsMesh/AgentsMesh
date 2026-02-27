"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { extensionApi, InstalledSkill, InstalledMcpServer } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { Plus } from "lucide-react";
import { SkillCard } from "./capabilities/SkillCard";
import { McpServerCard } from "./capabilities/McpServerCard";
import { AddSkillDialog } from "./capabilities/AddSkillDialog";
import { AddMcpServerDialog } from "./capabilities/AddMcpServerDialog";
import { EditMcpEnvVarsDialog } from "./capabilities/EditMcpEnvVarsDialog";

interface CapabilitiesTabProps {
  repositoryId: number;
}

export function CapabilitiesTab({ repositoryId }: CapabilitiesTabProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();

  const [orgSkills, setOrgSkills] = useState<InstalledSkill[]>([]);
  const [userSkills, setUserSkills] = useState<InstalledSkill[]>([]);
  const [orgMcpServers, setOrgMcpServers] = useState<InstalledMcpServer[]>([]);
  const [userMcpServers, setUserMcpServers] = useState<InstalledMcpServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddSkill, setShowAddSkill] = useState<"org" | "user" | null>(null);
  const [showAddMcp, setShowAddMcp] = useState<"org" | "user" | null>(null);
  const [editingMcp, setEditingMcp] = useState<InstalledMcpServer | null>(null);

  // Check if current user is org admin/owner
  const isAdmin = currentOrg?.role === "owner" || currentOrg?.role === "admin";

  // Confirm dialog for uninstall operations
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
      console.error("Failed to load skills:", error);
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
      console.error("Failed to load MCP servers:", error);
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToLoadMcpServers")));
    }
  }, [repositoryId, t]);

  useEffect(() => {
    const mounted = { current: true };
    const load = async () => {
      setLoading(true);
      await Promise.all([loadSkills(mounted), loadMcpServers(mounted)]);
      if (mounted.current) {
        setLoading(false);
      }
    };
    load();
    return () => { mounted.current = false; };
  }, [loadSkills, loadMcpServers]);

  const handleToggleSkill = useCallback(
    async (skill: InstalledSkill) => {
      try {
        await extensionApi.updateSkill(repositoryId, skill.id, { is_enabled: !skill.is_enabled });
        await loadSkills();
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate")));
      }
    },
    [repositoryId, loadSkills, t]
  );

  const handleDeleteSkill = useCallback(
    async (skill: InstalledSkill) => {
      const confirmed = await confirm({
        title: t("extensions.confirmUninstallSkill"),
        description: t("extensions.uninstallSkillDescription", { name: skill.slug }),
        variant: "destructive",
        confirmText: t("extensions.uninstall"),
        cancelText: t("extensions.cancel"),
      });
      if (!confirmed) return;
      try {
        await extensionApi.uninstallSkill(repositoryId, skill.id);
        toast.success(t("extensions.uninstalled"));
        await loadSkills();
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUninstall")));
      }
    },
    [repositoryId, loadSkills, t, confirm]
  );

  const handleToggleMcp = useCallback(
    async (mcp: InstalledMcpServer) => {
      try {
        await extensionApi.updateMcpServer(repositoryId, mcp.id, { is_enabled: !mcp.is_enabled });
        await loadMcpServers();
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate")));
      }
    },
    [repositoryId, loadMcpServers, t]
  );

  const handleDeleteMcp = useCallback(
    async (mcp: InstalledMcpServer) => {
      const confirmed = await confirm({
        title: t("extensions.confirmUninstallMcp"),
        description: t("extensions.uninstallMcpDescription", { name: mcp.name || mcp.slug }),
        variant: "destructive",
        confirmText: t("extensions.uninstall"),
        cancelText: t("extensions.cancel"),
      });
      if (!confirmed) return;
      try {
        await extensionApi.uninstallMcpServer(repositoryId, mcp.id);
        toast.success(t("extensions.uninstalled"));
        await loadMcpServers();
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUninstall")));
      }
    },
    [repositoryId, loadMcpServers, t, confirm]
  );

  if (loading) {
    return (
      <div className="p-8 text-center">
        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto"></div>
      </div>
    );
  }

  return (
    <Tabs defaultValue="skills">
      <TabsList>
        <TabsTrigger value="skills">{t("extensions.skills")}</TabsTrigger>
        <TabsTrigger value="mcp">{t("extensions.mcpServers")}</TabsTrigger>
      </TabsList>

      <TabsContent value="skills">
        <div className="space-y-6">
          {/* Organization Installed Skills */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                {t("extensions.orgInstalled")}
              </h3>
              {isAdmin && (
                <Button variant="outline" size="sm" onClick={() => setShowAddSkill("org")}>
                  <Plus className="w-4 h-4 mr-1" />
                  {t("extensions.add")}
                </Button>
              )}
            </div>
            {orgSkills.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4">{t("extensions.noSkillsInstalled")}</p>
            ) : (
              <div className="space-y-2">
                {orgSkills.map((skill) => (
                  <SkillCard
                    key={skill.id}
                    skill={skill}
                    canManage={isAdmin}
                    onToggle={handleToggleSkill}
                    onDelete={handleDeleteSkill}
                  />
                ))}
              </div>
            )}
          </section>

          {/* User Installed Skills */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                {t("extensions.myInstalled")}
              </h3>
              <Button variant="outline" size="sm" onClick={() => setShowAddSkill("user")}>
                <Plus className="w-4 h-4 mr-1" />
                {t("extensions.add")}
              </Button>
            </div>
            {userSkills.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4">{t("extensions.noSkillsInstalled")}</p>
            ) : (
              <div className="space-y-2">
                {userSkills.map((skill) => (
                  <SkillCard
                    key={skill.id}
                    skill={skill}
                    canManage={true}
                    onToggle={handleToggleSkill}
                    onDelete={handleDeleteSkill}
                  />
                ))}
              </div>
            )}
          </section>
        </div>
      </TabsContent>

      <TabsContent value="mcp">
        <div className="space-y-6">
          {/* Organization Installed MCP Servers */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                {t("extensions.orgInstalled")}
              </h3>
              {isAdmin && (
                <Button variant="outline" size="sm" onClick={() => setShowAddMcp("org")}>
                  <Plus className="w-4 h-4 mr-1" />
                  {t("extensions.add")}
                </Button>
              )}
            </div>
            {orgMcpServers.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4">{t("extensions.noMcpServersInstalled")}</p>
            ) : (
              <div className="space-y-2">
                {orgMcpServers.map((mcp) => (
                  <McpServerCard
                    key={mcp.id}
                    mcpServer={mcp}
                    canManage={isAdmin}
                    onToggle={handleToggleMcp}
                    onDelete={handleDeleteMcp}
                    onEditEnvVars={setEditingMcp}
                  />
                ))}
              </div>
            )}
          </section>

          {/* User Installed MCP Servers */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                {t("extensions.myInstalled")}
              </h3>
              <Button variant="outline" size="sm" onClick={() => setShowAddMcp("user")}>
                <Plus className="w-4 h-4 mr-1" />
                {t("extensions.add")}
              </Button>
            </div>
            {userMcpServers.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4">{t("extensions.noMcpServersInstalled")}</p>
            ) : (
              <div className="space-y-2">
                {userMcpServers.map((mcp) => (
                  <McpServerCard
                    key={mcp.id}
                    mcpServer={mcp}
                    canManage={true}
                    onToggle={handleToggleMcp}
                    onDelete={handleDeleteMcp}
                    onEditEnvVars={setEditingMcp}
                  />
                ))}
              </div>
            )}
          </section>
        </div>
      </TabsContent>

      {/* Dialogs */}
      {showAddSkill && (
        <AddSkillDialog
          repositoryId={repositoryId}
          scope={showAddSkill}
          open={true}
          onOpenChange={(open) => { if (!open) setShowAddSkill(null); }}
          onInstalled={() => {
            setShowAddSkill(null);
            loadSkills();
          }}
          installedSlugs={new Set([...orgSkills, ...userSkills].map((s) => s.slug))}
        />
      )}

      {showAddMcp && (
        <AddMcpServerDialog
          repositoryId={repositoryId}
          scope={showAddMcp}
          open={true}
          onOpenChange={(open) => { if (!open) setShowAddMcp(null); }}
          onInstalled={() => {
            setShowAddMcp(null);
            loadMcpServers();
          }}
        />
      )}

      {/* Edit MCP Env Vars Dialog */}
      {editingMcp && (
        <EditMcpEnvVarsDialog
          repositoryId={repositoryId}
          mcpServer={editingMcp}
          open={true}
          onOpenChange={(open) => { if (!open) setEditingMcp(null); }}
          onUpdated={() => {
            setEditingMcp(null);
            loadMcpServers();
          }}
        />
      )}

      {/* Uninstall Confirm Dialog */}
      <ConfirmDialog {...confirmDialogProps} />
    </Tabs>
  );
}
