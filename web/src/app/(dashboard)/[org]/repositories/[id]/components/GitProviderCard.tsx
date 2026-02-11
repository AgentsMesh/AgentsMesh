"use client";

import { RepositoryData } from "@/lib/api";
import { useTranslations } from "next-intl";

interface GitProviderCardProps {
  repository: RepositoryData;
}

export function GitProviderCard({ repository }: GitProviderCardProps) {
  const t = useTranslations();

  return (
    <div className="border border-border rounded-lg p-6">
      <h3 className="font-semibold mb-4">{t("repositories.detail.gitProvider")}</h3>
      <dl className="space-y-3">
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.type")}</dt>
          <dd className="font-medium capitalize">{repository.provider_type}</dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.baseUrl")}</dt>
          <dd className="font-medium">{repository.provider_base_url}</dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">{t("repositories.detail.visibility")}</dt>
          <dd className="font-medium capitalize">{repository.visibility}</dd>
        </div>
      </dl>
    </div>
  );
}
