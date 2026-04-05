"use client";

import { Button } from "@/components/ui/button";
import { Server, Key, Star, Check, Edit2, Trash2, Plus } from "lucide-react";
import type { CredentialsSectionProps } from "./types";

/**
 * CredentialsSection - Displays and manages credential profiles
 *
 * Shows RunnerHost as the first option and custom credential profiles below.
 * Allows setting default, editing, and deleting profiles.
 */
export function CredentialsSection({
  isRunnerHostDefault,
  credentialProfiles,
  onSetRunnerHostDefault,
  onSetDefault,
  onEdit,
  onDelete,
  onAdd,
  t,
}: CredentialsSectionProps) {
  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center gap-2 mb-4">
        <Key className="w-5 h-5 text-muted-foreground" />
        <h3 className="text-lg font-semibold">{t("settings.agentConfig.credentials.title")}</h3>
      </div>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.agentConfig.credentials.description")}
      </p>

      <div className="space-y-2">
        {/* RunnerHost - always shown as first option */}
        <div className="flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50">
          <div className="flex items-center gap-3">
            <Server className="w-4 h-4 text-muted-foreground" />
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium">RunnerHost</span>
                {isRunnerHostDefault && (
                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                    <Star className="w-3 h-3 mr-0.5" />
                    {t("settings.agentCredentials.default")}
                  </span>
                )}
              </div>
              <div className="text-xs text-muted-foreground">
                {t("settings.agentCredentials.runnerHostHint")}
              </div>
            </div>
          </div>
          {!isRunnerHostDefault && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onSetRunnerHostDefault}
              title={t("settings.agentCredentials.setAsDefault")}
            >
              <Check className="w-4 h-4" />
            </Button>
          )}
        </div>

        {/* Custom credential profiles */}
        {credentialProfiles.map((profile) => (
          <div
            key={profile.id}
            className="flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50"
          >
            <div className="flex items-center gap-3">
              <Key className="w-4 h-4 text-muted-foreground" />
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium">{profile.name}</span>
                  {profile.is_default && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                      <Star className="w-3 h-3 mr-0.5" />
                      {t("settings.agentCredentials.default")}
                    </span>
                  )}
                </div>
                <div className="text-xs text-muted-foreground">
                  {profile.configured_fields?.length
                    ? `${t("settings.agentCredentials.configured")}: ${profile.configured_fields.map((f) => {
                        const key = `settings.agentCredentials.fields.${f}`;
                        const translated = t(key);
                        return translated !== key ? translated : f;
                      }).join(", ")}`
                    : t("settings.agentCredentials.notConfigured")}
                </div>
              </div>
            </div>
            <div className="flex items-center gap-1">
              {!profile.is_default && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onSetDefault(profile.id)}
                  title={t("settings.agentCredentials.setAsDefault")}
                >
                  <Check className="w-4 h-4" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onEdit(profile)}
              >
                <Edit2 className="w-4 h-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDelete(profile.id)}
                className="text-destructive hover:text-destructive"
              >
                <Trash2 className="w-4 h-4" />
              </Button>
            </div>
          </div>
        ))}

        {/* Add credential button */}
        <Button
          variant="outline"
          size="sm"
          onClick={onAdd}
          className="mt-2"
        >
          <Plus className="w-4 h-4 mr-1" />
          {t("settings.agentCredentials.addProfile")}
        </Button>
      </div>
    </div>
  );
}

export default CredentialsSection;
