"use client";

import React, { useEffect, useState, useCallback } from "react";
import { Bell, BellOff, Check, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { usePushNotifications } from "@/components/pwa";
import { useTranslations } from "next-intl";
import { notificationApi, type NotificationPreference } from "@/lib/api";

// Available notification sources with i18n keys
const NOTIFICATION_SOURCES = [
  { source: "channel:message", labelKey: "settings.notifications.channelMessage", descKey: "settings.notifications.channelMessageDesc" },
  { source: "channel:mention", labelKey: "settings.notifications.channelMention", descKey: "settings.notifications.channelMentionDesc" },
  { source: "terminal:osc", labelKey: "settings.notifications.terminalOsc", descKey: "settings.notifications.terminalOscDesc" },
  { source: "task:completed", labelKey: "settings.notifications.taskCompleted", descKey: "settings.notifications.taskCompletedDesc" },
] as const;

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

      {/* Server-side notification preferences (toast / browser per source) */}
      <ServerNotificationPreferences />
    </div>
  );
}

// Channel label mapping for known delivery channels
const CHANNEL_LABELS: Record<string, string> = {
  toast: "Toast",
  browser: "Browser",
  apns: "Push (Mobile)",
  email: "Email",
};

/**
 * Server-synced notification preferences: mute / channels per source.
 */
function ServerNotificationPreferences() {
  const t = useTranslations();
  const [prefs, setPrefs] = useState<NotificationPreference[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchPrefs = useCallback(async () => {
    try {
      const res = await notificationApi.getPreferences();
      setPrefs(res.preferences || []);
    } catch {
      // Silently fail — user might not have org context yet
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPrefs();
  }, [fetchPrefs]);

  const getPref = (source: string): NotificationPreference => {
    const found = prefs.find((p) => p.source === source && !p.entity_id);
    return found ?? { source, is_muted: false, channels: { toast: true, browser: true } };
  };

  const handleMuteToggle = async (source: string, muted: boolean) => {
    const current = getPref(source);
    const updated = { ...current, is_muted: muted };
    setPrefs((prev) => {
      const idx = prev.findIndex((p) => p.source === source && !p.entity_id);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = updated;
        return next;
      }
      return [...prev, updated];
    });
    try {
      await notificationApi.setPreference(updated);
    } catch {
      fetchPrefs();
    }
  };

  const handleChannelToggle = async (source: string, channel: string, value: boolean) => {
    const current = getPref(source);
    const updated = {
      ...current,
      channels: { ...current.channels, [channel]: value },
    };
    setPrefs((prev) => {
      const idx = prev.findIndex((p) => p.source === source && !p.entity_id);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = updated;
        return next;
      }
      return [...prev, updated];
    });
    try {
      await notificationApi.setPreference(updated);
    } catch {
      fetchPrefs();
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h4 className="font-medium text-sm text-muted-foreground">
        {t("settings.notifications.deliveryPreferences")}
      </h4>

      <div className="space-y-3">
        {NOTIFICATION_SOURCES.map(({ source, labelKey, descKey }) => {
          const pref = getPref(source);
          return (
            <div key={source} className="p-3 rounded-lg border space-y-2">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="cursor-pointer font-medium">{t(labelKey)}</Label>
                  <p className="text-xs text-muted-foreground">{t(descKey)}</p>
                </div>
                <div className="flex items-center gap-1">
                  {pref.is_muted && (
                    <BellOff className="w-3.5 h-3.5 text-muted-foreground" />
                  )}
                  <Switch
                    checked={!pref.is_muted}
                    onCheckedChange={(checked) => handleMuteToggle(source, !checked)}
                  />
                </div>
              </div>
              {!pref.is_muted && pref.channels && (
                <div className="flex items-center gap-4 pl-1">
                  {Object.entries(pref.channels).map(([ch, enabled]) => (
                    <label key={ch} className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
                      <Switch className="scale-75" checked={enabled} onCheckedChange={(v) => handleChannelToggle(source, ch, v)} />
                      {CHANNEL_LABELS[ch] ?? ch}
                    </label>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
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
