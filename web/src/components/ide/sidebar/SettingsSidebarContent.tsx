"use client";

import React, { useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { getAgentService } from "@/lib/wasm-core";
import type { AgentData } from "@/lib/api";
import {
  Settings,
  Users,
  Bot,
  CreditCard,
  GitBranch,
  Bell,
  ChevronDown,
  ChevronRight,
  Sparkles,
  KeyRound,
  Puzzle,
  BarChart3,
} from "lucide-react";

interface SettingsSidebarContentProps {
  className?: string;
}

type SettingsScope = "personal" | "organization";

interface TabItem {
  id: string;
  labelKey?: string;
  label?: string;
  icon: typeof Settings;
  children?: TabItem[];
}

export function SettingsSidebarContent({ className }: SettingsSidebarContentProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const currentOrg = useCurrentOrg();
  const t = useTranslations();

  const currentScope: SettingsScope = (searchParams.get("scope") as SettingsScope) || "personal";
  const currentTab = searchParams.get("tab") || "general";

  const [expandedSubSections, setExpandedSubSections] = useState<Record<string, boolean>>({
    "agent-config": currentTab.startsWith("agents/"),
  });

  useEffect(() => {
    setExpandedSubSections((prev) => ({
      ...prev,
      "agent-config": currentTab.startsWith("agents/") || prev["agent-config"],
    }));
  }, [currentTab]);

  const [agents, setAgents] = useState<AgentData[]>([]);

  useEffect(() => {
    const fetchAgents = async () => {
      try {
        const response = JSON.parse(await getAgentService().list_agents());
        const merged = [
          ...(response.builtin_agents || []),
          ...(response.custom_agents || []),
        ];
        setAgents(merged);
      } catch (error) {
        console.error("Failed to fetch agents:", error);
      }
    };
    fetchAgents();
  }, []);

  const isOrgAdminOrOwner = currentOrg?.role === "owner" || currentOrg?.role === "admin";

  const orgSettingsTabs: TabItem[] = [
    { id: "general", labelKey: "ide.sidebar.settings.tabs.general", icon: Settings },
    { id: "members", labelKey: "ide.sidebar.settings.tabs.members", icon: Users },
    { id: "extensions", labelKey: "ide.sidebar.settings.tabs.extensions", icon: Puzzle },
    { id: "api-keys", labelKey: "ide.sidebar.settings.tabs.apiKeys", icon: KeyRound },
    ...(isOrgAdminOrOwner
      ? [{ id: "usage", labelKey: "ide.sidebar.settings.tabs.usage", icon: BarChart3 }]
      : []),
    { id: "billing", labelKey: "ide.sidebar.settings.tabs.billing", icon: CreditCard },
  ];

  const personalSettingsTabs: TabItem[] = [
    { id: "general", labelKey: "ide.sidebar.settings.tabs.general", icon: Settings },
    { id: "git", labelKey: "settings.personal.tabs.git", icon: GitBranch },
    {
      id: "agent-config",
      labelKey: "settings.personal.tabs.agentConfig",
      icon: Sparkles,
      children: agents.map((agent) => ({
        id: `agents/${agent.slug}`,
        label: agent.name,
        icon: Bot,
      })),
    },
    { id: "notifications", labelKey: "settings.personal.tabs.notifications", icon: Bell },
  ];

  const toggleSubSection = (sectionId: string) => {
    setExpandedSubSections((prev) => ({ ...prev, [sectionId]: !prev[sectionId] }));
  };

  const navigate = (scope: SettingsScope, tabId: string) => {
    router.push(`/${currentOrg?.slug}/settings?scope=${scope}&tab=${tabId}`);
  };

  const renderTabItem = (scope: SettingsScope, tab: TabItem, depth = 0): React.ReactNode => {
    const TabIcon = tab.icon;
    const hasChildren = tab.children && tab.children.length > 0;
    const isSubSectionExpanded = expandedSubSections[tab.id];
    const isActive = currentScope === scope && currentTab === tab.id;
    const isChildActive =
      hasChildren && tab.children?.some((child) => currentScope === scope && currentTab === child.id);
    const paddingLeft = depth === 0 ? "pl-3" : "pl-8";

    if (hasChildren) {
      return (
        <div key={tab.id}>
          <button
            className={cn(
              "w-full flex items-center gap-2 pr-3 py-1.5 text-left transition-colors rounded-md",
              paddingLeft,
              isChildActive ? "text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
            )}
            onClick={() => toggleSubSection(tab.id)}
          >
            {isSubSectionExpanded ? (
              <ChevronDown className="w-4 h-4 flex-shrink-0" />
            ) : (
              <ChevronRight className="w-4 h-4 flex-shrink-0" />
            )}
            <TabIcon className="w-4 h-4 flex-shrink-0" />
            <span className={cn("text-sm truncate", isChildActive && "font-medium")}>
              {tab.label || (tab.labelKey && t(tab.labelKey))}
            </span>
          </button>
          {isSubSectionExpanded && (
            <div className="ml-4 border-l border-border">
              {tab.children?.map((child) => renderTabItem(scope, child, depth + 1))}
            </div>
          )}
        </div>
      );
    }

    return (
      <button
        key={tab.id}
        className={cn(
          "w-full flex items-center gap-2 pr-3 py-1.5 text-left transition-colors rounded-md",
          paddingLeft,
          isActive ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
        )}
        onClick={() => navigate(scope, tab.id)}
      >
        <TabIcon className={cn("w-4 h-4 flex-shrink-0", isActive && "text-primary")} />
        <span className={cn("text-sm truncate", isActive && "font-medium")}>
          {tab.label || (tab.labelKey && t(tab.labelKey))}
        </span>
      </button>
    );
  };

  const activeTabs = currentScope === "personal" ? personalSettingsTabs : orgSettingsTabs;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Scope Tabs — visible, replaces query-param-driven toggle */}
      <div role="tablist" className="grid grid-cols-2 gap-1 p-2 border-b border-border">
        {(["personal", "organization"] as const).map((scope) => (
          <button
            key={scope}
            role="tab"
            aria-selected={currentScope === scope}
            onClick={() => navigate(scope, "general")}
            className={cn(
              "px-3 py-1.5 text-sm rounded-md transition-colors",
              currentScope === scope
                ? "bg-primary text-primary-foreground shadow-xs"
                : "text-muted-foreground hover:bg-muted hover:text-foreground",
            )}
          >
            {t(
              scope === "personal"
                ? "ide.sidebar.settings.scopePersonal"
                : "ide.sidebar.settings.scopeOrg",
            )}
          </button>
        ))}
      </div>

      {/* Tab tree for the active scope */}
      <div className="flex-1 overflow-y-auto p-2 space-y-0.5">
        {activeTabs.map((tab) => renderTabItem(currentScope, tab))}
      </div>

      {/* Organization info at bottom (only meaningful when on org scope) */}
      {currentOrg && currentScope === "organization" && (
        <div className="border-t border-border px-3 py-3">
          <div className="text-xs text-muted-foreground mb-1">{t("ide.sidebar.settings.currentOrg")}</div>
          <div className="text-sm font-medium truncate">{currentOrg.name}</div>
          <div className="text-xs text-muted-foreground truncate">/{currentOrg.slug}</div>
        </div>
      )}
    </div>
  );
}

export default SettingsSidebarContent;
