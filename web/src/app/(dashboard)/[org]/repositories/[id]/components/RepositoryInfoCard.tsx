"use client";

import { RepositoryData } from "@/lib/api";
import { useTranslations } from "next-intl";

interface RepositoryInfoCardProps {
  repository: RepositoryData;
}

export function RepositoryInfoCard({ repository }: RepositoryInfoCardProps) {
  const t = useTranslations();

  return (
    <div className="border border-border rounded-lg p-6">
      <h3 className="font-semibold mb-4">{t("repositories.detail.repoDetails")}</h3>
      <dl className="space-y-3">
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.name")}</dt>
          <dd className="font-medium">{repository.name}</dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.slug")}</dt>
          <dd className="font-medium">{repository.slug}</dd>
        </div>
        {repository.http_clone_url && (
          <div>
            <dt className="text-sm text-muted-foreground">{t("repositories.detail.httpCloneUrl")}</dt>
            <dd className="font-medium text-sm break-all">{repository.http_clone_url}</dd>
          </div>
        )}
        {repository.ssh_clone_url && (
          <div>
            <dt className="text-sm text-muted-foreground">{t("repositories.detail.sshCloneUrl")}</dt>
            <dd className="font-medium text-sm break-all">{repository.ssh_clone_url}</dd>
          </div>
        )}
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.defaultBranch")}</dt>
          <dd className="font-medium">{repository.default_branch}</dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.ticketPrefix")}</dt>
          <dd className="font-medium">{repository.ticket_prefix || "-"}</dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.status")}</dt>
          <dd>
            <span
              className={`inline-flex px-2 py-0.5 text-xs rounded ${
                repository.is_active
                  ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                  : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
              }`}
            >
              {repository.is_active ? t("repositories.detail.active") : t("repositories.inactive")}
            </span>
          </dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.created")}</dt>
          <dd className="font-medium">
            {new Date(repository.created_at).toLocaleString()}
          </dd>
        </div>
      </dl>
    </div>
  );
}
