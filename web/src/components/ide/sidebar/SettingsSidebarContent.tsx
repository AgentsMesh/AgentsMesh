"use client";

import React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import {
  Settings,
  Users,
  Bot,
  Server,
  GitBranch,
  Bell,
  CreditCard,
} from "lucide-react";

interface SettingsSidebarContentProps {
  className?: string;
}

// Settings tabs configuration
const settingsTabs = [
  { id: "general", label: "General", icon: Settings, description: "Organization details" },
  { id: "members", label: "Members", icon: Users, description: "Team members and roles" },
  { id: "agents", label: "Agents", icon: Bot, description: "AI agent types" },
  { id: "runners", label: "Runners", icon: Server, description: "Runner management" },
  { id: "git-providers", label: "Git Providers", icon: GitBranch, description: "Git integrations" },
  { id: "notifications", label: "Notifications", icon: Bell, description: "Notification preferences" },
  { id: "billing", label: "Billing", icon: CreditCard, description: "Subscription and payments" },
];

export function SettingsSidebarContent({ className }: SettingsSidebarContentProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { currentOrg } = useAuthStore();

  // Get current tab from URL params
  const currentTab = searchParams.get("tab") || "general";

  // Handle tab click
  const handleTabClick = (tabId: string) => {
    router.push(`/${currentOrg?.slug}/settings?tab=${tabId}`);
  };

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="px-3 py-3 border-b border-border">
        <h3 className="text-sm font-semibold">Settings</h3>
        <p className="text-xs text-muted-foreground mt-0.5">
          Manage your organization
        </p>
      </div>

      {/* Settings navigation */}
      <div className="flex-1 overflow-y-auto py-2">
        {settingsTabs.map((tab) => {
          const Icon = tab.icon;
          const isActive = currentTab === tab.id;

          return (
            <button
              key={tab.id}
              className={cn(
                "w-full flex items-start gap-3 px-3 py-2 text-left transition-colors",
                isActive
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              )}
              onClick={() => handleTabClick(tab.id)}
            >
              <Icon className={cn(
                "w-4 h-4 mt-0.5 flex-shrink-0",
                isActive && "text-primary"
              )} />
              <div className="flex-1 min-w-0">
                <p className={cn(
                  "text-sm truncate",
                  isActive && "font-medium"
                )}>
                  {tab.label}
                </p>
                <p className="text-xs text-muted-foreground truncate">
                  {tab.description}
                </p>
              </div>
            </button>
          );
        })}
      </div>

      {/* Organization info */}
      {currentOrg && (
        <div className="border-t border-border px-3 py-3">
          <div className="text-xs text-muted-foreground mb-1">Current Organization</div>
          <div className="text-sm font-medium truncate">{currentOrg.name}</div>
          <div className="text-xs text-muted-foreground truncate">/{currentOrg.slug}</div>
        </div>
      )}
    </div>
  );
}

export default SettingsSidebarContent;
