"use client";

import { Button } from "@/components/ui/button";
import { Key, Star, Check, Edit2, Trash2 } from "lucide-react";
import type { CredentialProfileViewModel } from "./_shared/credentialViewModel";
import { getConfiguredKeys } from "./_shared/credentialViewModel";
import { getEnvKeyLabel } from "./AgentCredentialsSettings/credentialForms";

interface CredentialProfileRowProps {
  profile: CredentialProfileViewModel;
  agentSlug: string;
  onSetDefault: (profileId: number) => void | Promise<void>;
  onEdit: (profile: CredentialProfileViewModel) => void;
  onDelete: (profileId: number) => void | Promise<void>;
  t: (key: string) => string;
}

// Inner content of a credential-profile list row — name, default badge,
// configured-keys summary, and the set-default/edit/delete actions. Shared by
// CredentialsSection (AgentConfigPage) and AgentItem (AgentCredentialsSettings),
// which differ only in their row wrapper's layout classes.
export function CredentialProfileRow({
  profile,
  agentSlug,
  onSetDefault,
  onEdit,
  onDelete,
  t,
}: CredentialProfileRowProps) {
  const configured = getConfiguredKeys(profile);
  return (
    <>
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
            {configured.length
              ? `${t("settings.agentCredentials.configured")}: ${configured
                  .map((f) => getEnvKeyLabel(agentSlug, f, t))
                  .join(", ")}`
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
          title={t("common.edit")}
        >
          <Edit2 className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onDelete(profile.id)}
          title={t("common.delete")}
          className="text-destructive hover:text-destructive"
        >
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
    </>
  );
}

export default CredentialProfileRow;
