"use client";

import { Button } from "@/components/ui/button";
import { GitCredentialData, CredentialTypeValue, getCredentialTypeLabel } from "@/lib/api";
import { Trash2 } from "lucide-react";
import { CredentialTypeIcon } from "@/components/icons/GitProviderIcon";

export interface GitCredentialCardProps {
  credential: GitCredentialData;
  onDelete: () => void;
  t: (key: string) => string;
}

export function GitCredentialCard({
  credential,
  onDelete,
  t,
}: GitCredentialCardProps) {
  return (
    <div className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
      <div className="flex items-center gap-4">
        <div className="w-10 h-10 rounded-full bg-background flex items-center justify-center">
          <CredentialTypeIcon type={credential.credential_type} />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-medium">{credential.name}</span>
            <span className="px-2 py-0.5 text-xs bg-muted text-muted-foreground rounded">
              {getCredentialTypeLabel(credential.credential_type as CredentialTypeValue)}
            </span>
          </div>
          {credential.fingerprint && (
            <p className="text-xs text-muted-foreground font-mono">
              {credential.fingerprint}
            </p>
          )}
          {credential.host_pattern && (
            <p className="text-xs text-muted-foreground">
              {t("settings.gitSettings.credentials.hostPattern")}: {credential.host_pattern}
            </p>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={onDelete}
          className="text-destructive hover:text-destructive"
        >
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
