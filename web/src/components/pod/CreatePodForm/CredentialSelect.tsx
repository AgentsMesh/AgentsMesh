"use client";

import { Spinner } from "@/components/ui/spinner";
import type { CredentialProfileData } from "@/lib/api";
import { RUNNER_HOST_PROFILE_ID } from "../hooks";

interface CredentialSelectProps {
  profiles: CredentialProfileData[];
  selectedProfileId: number;
  onSelect: (profileId: number) => void;
  loading?: boolean;
  t: (key: string) => string;
}

/**
 * Credential profile selection dropdown component
 */
export function CredentialSelect({
  profiles,
  selectedProfileId,
  onSelect,
  loading,
  t,
}: CredentialSelectProps) {
  return (
    <div>
      <label
        htmlFor="credential-select"
        className="block text-sm font-medium mb-2"
      >
        {t("ide.createPod.selectCredential")}
      </label>
      {loading ? (
        <div className="flex items-center text-sm text-muted-foreground py-2">
          <Spinner size="sm" className="mr-2" />
          {t("common.loading")}
        </div>
      ) : (
        <>
          <select
            id="credential-select"
            className="w-full px-3 py-2 border border-border rounded-md bg-background"
            value={selectedProfileId}
            onChange={(e) => onSelect(Number(e.target.value))}
          >
            <option value={RUNNER_HOST_PROFILE_ID}>
              RunnerHost ({t("ide.createPod.runnerHostDescription")})
            </option>
            {profiles.map((profile) => (
              <option key={profile.id} value={profile.id}>
                {profile.name}
                {profile.is_default
                  ? ` (${t("settings.agentCredentials.default")})`
                  : ""}
              </option>
            ))}
          </select>
          <p className="text-xs text-muted-foreground mt-1">
            {selectedProfileId === RUNNER_HOST_PROFILE_ID
              ? t("ide.createPod.runnerHostHint")
              : t("ide.createPod.customCredentialHint")}
          </p>
        </>
      )}
    </div>
  );
}
