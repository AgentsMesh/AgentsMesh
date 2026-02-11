"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { LogOut, User, Mail } from "lucide-react";

export default function GeneralSettingsPage() {
  const router = useRouter();
  const t = useTranslations();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-foreground">
          {t("settings.personal.general.title")}
        </h1>
        <p className="text-muted-foreground">
          {t("settings.personal.general.description")}
        </p>
      </div>

      {/* Account Information */}
      <div className="border border-border rounded-lg p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">
          {t("settings.personal.general.accountInfo")}
        </h2>

        <div className="space-y-4">
          {/* Username */}
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

          {/* Email */}
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

      {/* Session Management */}
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
          className="flex items-center gap-2"
        >
          <LogOut className="w-4 h-4" />
          {t("settings.personal.general.logout")}
        </Button>
      </div>
    </div>
  );
}
