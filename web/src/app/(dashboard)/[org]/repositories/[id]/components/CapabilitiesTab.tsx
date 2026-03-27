"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { InstalledSkill, InstalledMcpServer } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Plus } from "lucide-react";
import { SkillCard } from "./capabilities/SkillCard";
import { McpServerCard } from "./capabilities/McpServerCard";
import { AddSkillDialog } from "./capabilities/AddSkillDialog";
import { AddMcpServerDialog } from "./capabilities/AddMcpServerDialog";
import { EditMcpEnvVarsDialog } from "./capabilities/EditMcpEnvVarsDialog";
import { useCapabilitiesData } from "./useCapabilitiesData";

interface CapabilitiesTabProps { repositoryId: number; }

function ExtensionSection<T extends InstalledSkill | InstalledMcpServer>({
  title, items, emptyMessage, canManage, onAdd, addLabel,
  renderCard,
}: {
  title: string; items: T[]; emptyMessage: string; canManage: boolean;
  onAdd: () => void; addLabel: string; renderCard: (item: T) => React.ReactNode;
}) {
  return (
    <section>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">{title}</h3>
        {canManage && (
          <Button variant="outline" size="sm" onClick={onAdd}>
            <Plus className="w-4 h-4 mr-1" />{addLabel}
          </Button>
        )}
      </div>
      {items.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4">{emptyMessage}</p>
      ) : (
        <div className="space-y-2">{items.map(renderCard)}</div>
      )}
    </section>
  );
}

export function CapabilitiesTab({ repositoryId }: CapabilitiesTabProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const isAdmin = currentOrg?.role === "owner" || currentOrg?.role === "admin";

  const {
    orgSkills, userSkills, orgMcpServers, userMcpServers, loading,
    loadSkills, loadMcpServers,
    handleToggleSkill, handleDeleteSkill, handleToggleMcp, handleDeleteMcp,
    confirmDialogProps,
  } = useCapabilitiesData(repositoryId);

  const [showAddSkill, setShowAddSkill] = useState<"org" | "user" | null>(null);
  const [showAddMcp, setShowAddMcp] = useState<"org" | "user" | null>(null);
  const [editingMcp, setEditingMcp] = useState<InstalledMcpServer | null>(null);

  if (loading) {
    return <div className="p-8 text-center"><div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto" /></div>;
  }

  return (
    <Tabs defaultValue="skills">
      <TabsList>
        <TabsTrigger value="skills">{t("extensions.skills")}</TabsTrigger>
        <TabsTrigger value="mcp">{t("extensions.mcpServers")}</TabsTrigger>
      </TabsList>

      <TabsContent value="skills">
        <div className="space-y-6">
          <ExtensionSection title={t("extensions.orgInstalled")} items={orgSkills} emptyMessage={t("extensions.noSkillsInstalled")}
            canManage={isAdmin} addLabel={t("extensions.add")} onAdd={() => setShowAddSkill("org")}
            renderCard={(s) => <SkillCard key={s.id} skill={s} canManage={isAdmin} onToggle={handleToggleSkill} onDelete={handleDeleteSkill} />} />
          <ExtensionSection title={t("extensions.myInstalled")} items={userSkills} emptyMessage={t("extensions.noSkillsInstalled")}
            canManage={true} addLabel={t("extensions.add")} onAdd={() => setShowAddSkill("user")}
            renderCard={(s) => <SkillCard key={s.id} skill={s} canManage={true} onToggle={handleToggleSkill} onDelete={handleDeleteSkill} />} />
        </div>
      </TabsContent>

      <TabsContent value="mcp">
        <div className="space-y-6">
          <ExtensionSection title={t("extensions.orgInstalled")} items={orgMcpServers} emptyMessage={t("extensions.noMcpServersInstalled")}
            canManage={isAdmin} addLabel={t("extensions.add")} onAdd={() => setShowAddMcp("org")}
            renderCard={(m) => <McpServerCard key={m.id} mcpServer={m} canManage={isAdmin} onToggle={handleToggleMcp} onDelete={handleDeleteMcp} onEditEnvVars={setEditingMcp} />} />
          <ExtensionSection title={t("extensions.myInstalled")} items={userMcpServers} emptyMessage={t("extensions.noMcpServersInstalled")}
            canManage={true} addLabel={t("extensions.add")} onAdd={() => setShowAddMcp("user")}
            renderCard={(m) => <McpServerCard key={m.id} mcpServer={m} canManage={true} onToggle={handleToggleMcp} onDelete={handleDeleteMcp} onEditEnvVars={setEditingMcp} />} />
        </div>
      </TabsContent>

      {showAddSkill && <AddSkillDialog repositoryId={repositoryId} scope={showAddSkill} open={true}
        onOpenChange={(open) => { if (!open) setShowAddSkill(null); }}
        onInstalled={() => { setShowAddSkill(null); loadSkills(); }}
        installedSlugs={new Set([...orgSkills, ...userSkills].map((s) => s.slug))} />}
      {showAddMcp && <AddMcpServerDialog repositoryId={repositoryId} scope={showAddMcp} open={true}
        onOpenChange={(open) => { if (!open) setShowAddMcp(null); }}
        onInstalled={() => { setShowAddMcp(null); loadMcpServers(); }} />}
      {editingMcp && <EditMcpEnvVarsDialog repositoryId={repositoryId} mcpServer={editingMcp} open={true}
        onOpenChange={(open) => { if (!open) setEditingMcp(null); }}
        onUpdated={() => { setEditingMcp(null); loadMcpServers(); }} />}
      <ConfirmDialog {...confirmDialogProps} />
    </Tabs>
  );
}
