"use client";

import { Button } from "@/components/ui/button";
import type { RepositoryProviderData } from "@/lib/api/userRepositoryProviderTypes";
import { Settings, Trash2, TestTube } from "lucide-react";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";

export interface GitProviderCardProps {
  provider: RepositoryProviderData;
  onEdit: () => void;
  onDelete: () => void;
  onTestConnection: () => void;
  t: (key: string) => string;
}

export function GitProviderCard({
  provider,
  onEdit,
  onDelete,
  onTestConnection,
  t,
}: GitProviderCardProps) {
  const isDisabled = provider.is_active === false;
  return (
    <div
      data-testid="git-provider-card"
      data-provider-id={provider.id}
      data-is-active={provider.is_active === false ? "false" : "true"}
      className={`flex items-center justify-between p-4 rounded-lg border ${
        isDisabled ? "opacity-60 bg-muted/30" : "bg-muted/50"
      }`}
    >
      <div className="flex items-center gap-4">
        <div className="w-10 h-10 rounded-full bg-background flex items-center justify-center">
          <GitProviderIcon provider={provider.provider_type} />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-medium">{provider.name}</span>
            {provider.is_default && (
              <span className="px-2 py-0.5 text-xs bg-primary/10 text-primary rounded-full">
                {t("settings.gitSettings.providers.default")}
              </span>
            )}
            {isDisabled && (
              <span
                data-testid="git-provider-disabled-badge"
                className="px-2 py-0.5 text-xs bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 rounded-full"
              >
                {t("settings.gitSettings.providers.disabled")}
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">{provider.base_url}</p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={onTestConnection}
          title={t("settings.gitSettings.providers.test")}
        >
          <TestTube className="w-4 h-4" />
        </Button>
        <Button
          data-testid="git-provider-edit-button"
          variant="ghost"
          size="sm"
          onClick={onEdit}
        >
          <Settings className="w-4 h-4" />
        </Button>
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
