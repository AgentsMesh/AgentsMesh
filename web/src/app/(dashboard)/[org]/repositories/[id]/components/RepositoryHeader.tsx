"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { RepositoryData } from "@/lib/api";
import { useTranslations } from "next-intl";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";

interface RepositoryHeaderProps {
  repository: RepositoryData;
  onEdit: () => void;
  onDelete: () => void;
}

export function RepositoryHeader({ repository, onEdit, onDelete }: RepositoryHeaderProps) {
  const t = useTranslations();

  return (
    <>
      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div className="flex items-start gap-4">
          <div className="mt-1 text-muted-foreground">
            <GitProviderIcon provider={repository.provider_type} className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-foreground">{repository.name}</h1>
              {!repository.is_active && (
                <span className="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400 rounded">
                  {t("repositories.inactive")}
                </span>
              )}
              {repository.visibility === "private" && (
                <span className="px-2 py-0.5 text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 rounded">
                  {t("repositories.repository.private")}
                </span>
              )}
            </div>
            <p className="text-muted-foreground">{repository.slug}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={onEdit}>
            {t("common.edit")}
          </Button>
          <Button variant="destructive" onClick={onDelete}>
            {t("common.delete")}
          </Button>
        </div>
      </div>

      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-muted-foreground mb-6">
        <Link href="../repositories" className="hover:text-foreground">
          {t("repositories.title")}
        </Link>
        <span>/</span>
        <span className="text-foreground">{repository.name}</span>
      </div>
    </>
  );
}
