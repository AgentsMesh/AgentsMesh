"use client";

import React from "react";
import { Bell, BellOff, Check, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { usePushNotifications } from "@/components/pwa";
import { useTranslations } from "next-intl";

interface NotificationSettingsProps {
  className?: string;
}

export function NotificationSettings({ className }: NotificationSettingsProps) {
  const t = useTranslations();
  const {
    permission,
    subscription,
    preferences,
    isSupported,
    isLoading,
    error,
    requestPermission,
    subscribe,
    unsubscribe,
    updatePreferences,
  } = usePushNotifications();

  const handleEnableNotifications = async () => {
    const granted = await requestPermission();
    if (granted) {
      await subscribe();
    }
  };

  const handleDisableNotifications = async () => {
    await unsubscribe();
  };

  const isEnabled = permission === "granted" && subscription !== null;

  if (!isSupported) {
    return (
      <div className={cn("p-4 rounded-lg bg-muted/50", className)}>
        <div className="flex items-center gap-3 text-muted-foreground">
          <BellOff className="w-5 h-5" />
          <span>{t("settings.notifications.notSupported")}</span>
        </div>
      </div>
    );
  }

  return (
    <div className={cn("space-y-6", className)}>
      {/* Enable/Disable Section */}
      <div className="flex items-center justify-between p-4 rounded-lg border">
        <div className="flex items-center gap-3">
          {isEnabled ? (
            <div className="w-10 h-10 rounded-full bg-green-500/10 flex items-center justify-center">
              <Bell className="w-5 h-5 text-green-500 dark:text-green-400" />
            </div>
          ) : (
            <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
              <BellOff className="w-5 h-5 text-muted-foreground" />
            </div>
          )}
          <div>
            <p className="font-medium">{t("settings.notifications.title")}</p>
            <p className="text-sm text-muted-foreground">
              {isEnabled
                ? t("settings.notifications.enabled")
                : t("settings.notifications.disabled")}
            </p>
          </div>
        </div>

        <Button
          variant={isEnabled ? "outline" : "default"}
          onClick={isEnabled ? handleDisableNotifications : handleEnableNotifications}
          disabled={isLoading || permission === "denied"}
        >
          {isLoading ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : isEnabled ? (
            t("settings.notifications.disable")
          ) : permission === "denied" ? (
            t("settings.notifications.blocked")
          ) : (
            t("settings.notifications.enable")
          )}
        </Button>
      </div>

      {permission === "denied" && (
        <div className="p-4 rounded-lg bg-destructive/10 text-destructive text-sm">
          <p className="font-medium">{t("settings.notifications.blockedTitle")}</p>
          <p className="text-destructive/80">
            {t("settings.notifications.blockedHint")}
          </p>
        </div>
      )}

      {error && (
        <div className="p-4 rounded-lg bg-destructive/10 text-destructive text-sm">
          {error}
        </div>
      )}

      {/* Preferences Section */}
      {isEnabled && (
        <div className="space-y-4">
          <h4 className="font-medium text-sm text-muted-foreground">{t("settings.notifications.types")}</h4>

          <div className="space-y-3">
            <NotificationPreferenceItem
              id="pod-status"
              label={t("settings.notifications.podStatus")}
              description={t("settings.notifications.podStatusDesc")}
              checked={preferences.podStatus}
              onChange={(checked) => updatePreferences({ podStatus: checked })}
            />

            <NotificationPreferenceItem
              id="ticket-assigned"
              label={t("settings.notifications.ticketAssigned")}
              description={t("settings.notifications.ticketAssignedDesc")}
              checked={preferences.ticketAssigned}
              onChange={(checked) => updatePreferences({ ticketAssigned: checked })}
            />

            <NotificationPreferenceItem
              id="ticket-updated"
              label={t("settings.notifications.ticketUpdated")}
              description={t("settings.notifications.ticketUpdatedDesc")}
              checked={preferences.ticketUpdated}
              onChange={(checked) => updatePreferences({ ticketUpdated: checked })}
            />

            <NotificationPreferenceItem
              id="runner-offline"
              label={t("settings.notifications.runnerOffline")}
              description={t("settings.notifications.runnerOfflineDesc")}
              checked={preferences.runnerOffline}
              onChange={(checked) => updatePreferences({ runnerOffline: checked })}
            />
          </div>
        </div>
      )}

      {/* Status indicator */}
      {isEnabled && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Check className="w-4 h-4 text-green-500 dark:text-green-400" />
          <span>{t("settings.notifications.active")}</span>
        </div>
      )}
    </div>
  );
}

interface NotificationPreferenceItemProps {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

function NotificationPreferenceItem({
  id,
  label,
  description,
  checked,
  onChange,
}: NotificationPreferenceItemProps) {
  return (
    <div className="flex items-center justify-between p-3 rounded-lg border">
      <div className="space-y-0.5">
        <Label htmlFor={id} className="cursor-pointer">
          {label}
        </Label>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
    </div>
  );
}

export default NotificationSettings;
