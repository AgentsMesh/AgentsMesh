"use client";

import React, { useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { agentApi, AgentTypeData } from "@/lib/api/agent";
import {
  Settings,
  Users,
  Bot,
  CreditCard,
  User,
  GitBranch,
  Bell,
  Building2,
  ChevronDown,
  ChevronRight,
  Sparkles,
} from "lucide-react";

interface SettingsSidebarContentProps {
  className?: string;
}

type SettingsScope = "organization" | "personal";

// Tab item with optional children for three-level menu
interface TabItem {
  id: string;
  labelKey?: string;
  label?: string;  // Direct label (for dynamic items)
  icon: typeof Settings;
  children?: TabItem[];
}

export function SettingsSidebarContent({ className }: SettingsSidebarContentProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { currentOrg } = useAuthStore();
  const t = useTranslations();

  // Get current scope and tab from URL params - default to personal scope
  const currentScope: SettingsScope = (searchParams.get("scope") as SettingsScope) || "personal";
  const currentTab = searchParams.get("tab") || "general";

  // Track expanded sections - expand personal by default
  const [expandedSections, setExpandedSections] = useState<Record<SettingsScope, boolean>>({
    personal: true,
    organization: currentScope === "organization",
  });

  // Track expanded sub-sections (for three-level menu)
  const [expandedSubSections, setExpandedSubSections] = useState<Record<string, boolean>>({
    "agent-config": currentTab.startsWith("agents/"),
  });

  // Agent types from backend
  const [agentTypes, setAgentTypes] = useState<AgentTypeData[]>([]);

  // Fetch agent types on mount
  useEffect(() => {
    const fetchAgentTypes = async () => {
      try {
        const response = await agentApi.listTypes();
        setAgentTypes(response.agent_types || []);
      } catch (error) {
        console.error("Failed to fetch agent types:", error);
      }
    };
    fetchAgentTypes();
  }, []);

  // Organization settings tabs (removed "agents" - now in personal settings)
  const orgSettingsTabs: TabItem[] = [
    { id: "general", labelKey: "ide.sidebar.settings.tabs.general", icon: Settings },
    { id: "members", labelKey: "ide.sidebar.settings.tabs.members", icon: Users },
    { id: "billing", labelKey: "ide.sidebar.settings.tabs.billing", icon: CreditCard },
  ];

  // Personal settings tabs with agent config as expandable section
  const personalSettingsTabs: TabItem[] = [
    { id: "general", labelKey: "ide.sidebar.settings.tabs.general", icon: Settings },
    { id: "git", labelKey: "settings.personal.tabs.git", icon: GitBranch },
    {
      id: "agent-config",
      labelKey: "settings.personal.tabs.agentConfig",
      icon: Sparkles,
      children: agentTypes.map(agent => ({
        id: `agents/${agent.slug}`,
        label: agent.name,
        icon: Bot,
      })),
    },
    { id: "notifications", labelKey: "settings.personal.tabs.notifications", icon: Bell },
  ];

  // Toggle section expansion
  const toggleSection = (scope: SettingsScope) => {
    setExpandedSections(prev => ({
      ...prev,
      [scope]: !prev[scope],
    }));
  };

  // Toggle sub-section expansion (for three-level menu)
  const toggleSubSection = (sectionId: string) => {
    setExpandedSubSections(prev => ({
      ...prev,
      [sectionId]: !prev[sectionId],
    }));
  };

  // Handle tab click
  const handleTabClick = (scope: SettingsScope, tabId: string) => {
    if (scope === "organization") {
      router.push(`/${currentOrg?.slug}/settings?scope=organization&tab=${tabId}`);
    } else {
      router.push(`/${currentOrg?.slug}/settings?scope=personal&tab=${tabId}`);
    }
    // Auto expand the section when clicking a tab
    setExpandedSections(prev => ({
      ...prev,
      [scope]: true,
    }));
  };

  // Render tab item (can be simple or expandable with children)
  const renderTabItem = (
    scope: SettingsScope,
    tab: TabItem,
    depth: number = 0
  ) => {
    const TabIcon = tab.icon;
    const hasChildren = tab.children && tab.children.length > 0;
    const isSubSectionExpanded = expandedSubSections[tab.id];

    // Check if this tab or any of its children is active
    const isActive = currentScope === scope && currentTab === tab.id;
    const isChildActive = hasChildren && tab.children?.some(
      child => currentScope === scope && currentTab === child.id
    );

    const paddingLeft = depth === 0 ? "pl-4" : "pl-8";

    if (hasChildren) {
      // Render expandable sub-section
      return (
        <div key={tab.id}>
          <button
            className={cn(
              "w-full flex items-center gap-2 pr-3 py-1.5 text-left transition-colors",
              paddingLeft,
              isChildActive
                ? "text-foreground"
                : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
            )}
            onClick={() => toggleSubSection(tab.id)}
          >
            {isSubSectionExpanded ? (
              <ChevronDown className="w-3 h-3 text-muted-foreground flex-shrink-0" />
            ) : (
              <ChevronRight className="w-3 h-3 text-muted-foreground flex-shrink-0" />
            )}
            <TabIcon className={cn(
              "w-4 h-4 flex-shrink-0",
              isChildActive && "text-primary"
            )} />
            <span className={cn(
              "text-sm truncate",
              isChildActive && "font-medium"
            )}>
              {tab.label || (tab.labelKey && t(tab.labelKey))}
            </span>
          </button>

          {/* Render children */}
          {isSubSectionExpanded && (
            <div className="ml-4 border-l border-border/50">
              {tab.children?.map(child => renderTabItem(scope, child, depth + 1))}
            </div>
          )}
        </div>
      );
    }

    // Render simple tab
    return (
      <button
        key={tab.id}
        className={cn(
          "w-full flex items-center gap-2 pr-3 py-1.5 text-left transition-colors",
          paddingLeft,
          isActive
            ? "bg-muted text-foreground"
            : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
        )}
        onClick={() => handleTabClick(scope, tab.id)}
      >
        <TabIcon className={cn(
          "w-4 h-4 flex-shrink-0",
          isActive && "text-primary"
        )} />
        <span className={cn(
          "text-sm truncate",
          isActive && "font-medium"
        )}>
          {tab.label || (tab.labelKey && t(tab.labelKey))}
        </span>
      </button>
    );
  };

  // Render a collapsible section
  const renderSection = (
    scope: SettingsScope,
    titleKey: string,
    Icon: typeof Building2,
    tabs: TabItem[]
  ) => {
    const isExpanded = expandedSections[scope];
    const isCurrentScope = currentScope === scope;

    return (
      <div className="mb-1">
        {/* Section header */}
        <button
          className={cn(
            "w-full flex items-center gap-2 px-3 py-2 text-left transition-colors",
            "hover:bg-muted/50",
            isCurrentScope && "text-foreground"
          )}
          onClick={() => toggleSection(scope)}
        >
          {isExpanded ? (
            <ChevronDown className="w-4 h-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="w-4 h-4 text-muted-foreground" />
          )}
          <Icon className="w-4 h-4" />
          <span className="text-sm font-medium">{t(titleKey)}</span>
        </button>

        {/* Section items */}
        {isExpanded && (
          <div className="ml-4 border-l border-border">
            {tabs.map(tab => renderTabItem(scope, tab))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Settings navigation */}
      <div className="flex-1 overflow-y-auto py-2">
        {/* Personal settings section (on top) */}
        {renderSection(
          "personal",
          "ide.sidebar.settings.scopePersonal",
          User,
          personalSettingsTabs
        )}

        {/* Organization settings section */}
        {renderSection(
          "organization",
          "ide.sidebar.settings.scopeOrg",
          Building2,
          orgSettingsTabs
        )}
      </div>

      {/* Organization info at bottom */}
      {currentOrg && (
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
