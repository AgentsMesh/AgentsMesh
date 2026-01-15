"use client";

import { useSearchParams } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { LanguageSettings, NotificationSettings, AgentCredentialsSettings, AgentConfigPage, GitSettingsContent } from "@/components/settings";
import { GeneralSettings, MembersSettings, BillingSettings, RunnersSettings } from "@/components/settings/organization";
import { useTranslations } from "@/lib/i18n/client";

export default function SettingsPage() {
  const searchParams = useSearchParams();
  const scope = searchParams.get("scope") || "personal";
  const activeTab = searchParams.get("tab") || "general";
  const { currentOrg } = useAuthStore();
  const t = useTranslations();

  const renderContent = () => {
    // Personal settings
    if (scope === "personal") {
      // Handle agent config pages (agents/{slug})
      if (activeTab.startsWith("agents/")) {
        const agentSlug = activeTab.replace("agents/", "");
        return <AgentConfigPage agentSlug={agentSlug} />;
      }

      switch (activeTab) {
        case "general":
          return <PersonalGeneralSettings />;
        case "git":
          return <GitSettingsContent />;
        case "agent-credentials":
          return <PersonalAgentCredentialsSettings />;
        case "notifications":
          return <PersonalNotificationsSettings t={t} />;
        default:
          return <PersonalGeneralSettings />;
      }
    }

    // Organization settings
    switch (activeTab) {
      case "general":
        return <GeneralSettings org={currentOrg} t={t} />;
      case "members":
        return <MembersSettings t={t} />;
      case "runners":
        return <RunnersSettings t={t} />;
      case "billing":
        return <BillingSettings t={t} />;
      default:
        return <GeneralSettings org={currentOrg} t={t} />;
    }
  };

  return (
    <div className="h-full overflow-auto p-6">
      <div className="max-w-4xl">
        {renderContent()}
      </div>
    </div>
  );
}

// ===== Personal Settings Components =====

function PersonalGeneralSettings() {
  return (
    <div className="space-y-6">
      <LanguageSettings />
    </div>
  );
}

function PersonalAgentCredentialsSettings() {
  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <AgentCredentialsSettings />
      </div>
    </div>
  );
}

type TranslationFn = (key: string, params?: Record<string, string | number>) => string;

function PersonalNotificationsSettings({ t }: { t: TranslationFn }) {
  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.notifications.title")}</h2>
        <p className="text-sm text-muted-foreground mb-6">
          {t("settings.notifications.description")}
        </p>
        <NotificationSettings />
      </div>
    </div>
  );
}
