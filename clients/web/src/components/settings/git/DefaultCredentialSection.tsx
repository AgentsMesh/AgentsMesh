"use client";

import { Check } from "lucide-react";
import { CredentialType, CredentialTypeValue, getCredentialTypeLabel } from "@/lib/api";
import { CredentialTypeIcon } from "@/components/icons/GitProviderIcon";

export interface SelectableCredential {
  id: number | "runner_local";
  name: string;
  type: string;
  isDefault: boolean;
}

export interface DefaultCredentialSectionProps {
  credentials: SelectableCredential[];
  onSetDefault: (credentialId: number | null) => void;
  t: (key: string) => string;
}

export function DefaultCredentialSection({
  credentials,
  onSetDefault,
  t,
}: DefaultCredentialSectionProps) {
  return (
    <div className="border border-border rounded-lg p-6 mb-6">
      <h2 className="text-lg font-semibold mb-2">
        {t("settings.gitSettings.defaultCredential.title")}
      </h2>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.gitSettings.defaultCredential.description")}
      </p>

      <div className="space-y-2">
        {credentials.map((cred) => (
          <button
            key={cred.id}
            onClick={() =>
              onSetDefault(cred.id === "runner_local" ? null : (cred.id as number))
            }
            className={`w-full flex items-center gap-3 p-3 rounded-lg border transition-colors text-left ${
              cred.isDefault
                ? "border-primary bg-primary/5"
                : "border-border hover:bg-muted/50"
            }`}
          >
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center ${
                cred.isDefault ? "bg-primary text-primary-foreground" : "bg-muted"
              }`}
            >
              <CredentialTypeIcon type={cred.type} />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className="font-medium">{cred.name}</span>
                <span className="text-xs px-2 py-0.5 rounded bg-muted text-muted-foreground">
                  {getCredentialTypeLabel(cred.type as CredentialTypeValue)}
                </span>
              </div>
              {cred.type === CredentialType.RUNNER_LOCAL && (
                <p className="text-xs text-muted-foreground">
                  {t("settings.gitSettings.defaultCredential.runnerLocalHint")}
                </p>
              )}
            </div>
            {cred.isDefault && <Check className="w-5 h-5 text-primary" />}
          </button>
        ))}
      </div>
    </div>
  );
}
