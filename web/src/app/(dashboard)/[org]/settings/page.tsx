"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/button";
import { LanguageSettings, ThemeSettings, NotificationSettings, AgentCredentialsSettings, AgentConfigPage, GitSettingsContent } from "@/components/settings";
import { GeneralSettings, MembersSettings, BillingSettings, RunnersSettings, APIKeysSettings, ExtensionsSettings } from "@/components/settings/organization";
import { useTranslations } from "next-intl";
import { LogOut, User, Mail } from "lucide-react";

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
      case "extensions":
        return <ExtensionsSettings t={t} />;
      case "runners":
        return <RunnersSettings t={t} />;
      case "api-keys":
        return <APIKeysSettings t={t} />;
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
  const router = useRouter();
  const t = useTranslations();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  return (
    <div className="space-y-6">
      {/* Account Information */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">
          {t("settings.personal.general.accountInfo")}
        </h2>
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
              <User className="w-5 h-5 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">
                {t("settings.personal.general.username")}
              </p>
              <p className="font-medium">{user?.username || "-"}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
              <Mail className="w-5 h-5 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">
                {t("settings.personal.general.email")}
              </p>
              <p className="font-medium">{user?.email || "-"}</p>
            </div>
          </div>
        </div>
      </div>

      <LanguageSettings />
      <ThemeSettings />

      {/* Session / Logout */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-2">
          {t("settings.personal.general.session")}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.personal.general.sessionDescription")}
        </p>
        <Button
          variant="outline"
          onClick={handleLogout}
          className="flex items-center gap-2 text-destructive hover:text-destructive"
        >
          <LogOut className="w-4 h-4" />
          {t("settings.personal.general.logout")}
        </Button>
      </div>
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
