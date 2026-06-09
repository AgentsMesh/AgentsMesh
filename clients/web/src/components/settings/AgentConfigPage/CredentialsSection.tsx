"use client";

import { Button } from "@/components/ui/button";
import { Server, Key, Star, Check, Plus } from "lucide-react";
import type { CredentialsSectionProps } from "./types";
import { CredentialProfileRow } from "../CredentialProfileRow";

/**
 * CredentialsSection - Displays and manages credential bundles for an agent
 *
 * Shows a "no bundle" row first (Runner uses its native env) followed by
 * any custom credential-kind EnvBundles attached to this agent.
 * Allows setting default, editing, and deleting bundles.
 */
export function CredentialsSection({
  agentSlug,
  noPrimaryBundle,
  credentialProfiles,
  onClearPrimary,
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
        {/* "No bundle" — always shown as first option.
            Represents using the Runner's native env; selecting it clears the
            primary credential bundle for this agent. */}
        <div className="flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50">
          <div className="flex items-center gap-3">
            <Server className="w-4 h-4 text-muted-foreground" />
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium">{t("settings.agentCredentials.noBundleLabel")}</span>
                {noPrimaryBundle && (
                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                    <Star className="w-3 h-3 mr-0.5" />
                    {t("settings.agentCredentials.default")}
                  </span>
                )}
              </div>
              <div className="text-xs text-muted-foreground">
                {t("settings.agentCredentials.noBundleHint")}
              </div>
            </div>
          </div>
          {!noPrimaryBundle && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onClearPrimary}
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
            <CredentialProfileRow
              profile={profile}
              agentSlug={agentSlug}
              onSetDefault={onSetDefault}
              onEdit={onEdit}
              onDelete={onDelete}
              t={t}
            />
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
