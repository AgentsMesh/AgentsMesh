"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { repositoryApi } from "@/lib/api";
import type { RepositoryData } from "@/lib/api";
import { useTranslations } from "next-intl";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";
import { ImportRepositoryModal } from "@/components/ide/modals/ImportRepositoryModal";

export default function RepositoriesPage() {
  const { org: orgSlug } = useParams<{ org: string }>();
  const t = useTranslations();
  const [repositories, setRepositories] = useState<RepositoryData[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("");
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [showImportModal, setShowImportModal] = useState(false);

  const deleteDialog = useConfirmDialog({
    title: t("repositories.deleteDialog.title"),
    description: t("repositories.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const reposRes = await repositoryApi.list();
      setRepositories(reposRes.repositories || []);
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = useCallback(async (id: number) => {
    const confirmed = await deleteDialog.confirm();
    if (confirmed) {
      try {
        await repositoryApi.delete(id);
        setRepositories((prev) => prev.filter((r) => r.id !== id));
      } catch (error) {
        console.error("Failed to delete repository:", error);
      }
    }
  }, [deleteDialog]);

  const filteredRepositories = repositories.filter((repo) => {
    const matchesSearch =
      repo.name.toLowerCase().includes(filter.toLowerCase()) ||
      repo.full_path.toLowerCase().includes(filter.toLowerCase());
    const matchesProvider = !providerFilter || repo.provider_type === providerFilter;
    return matchesSearch && matchesProvider;
  });

  // Get unique provider types for filter
  const providerTypes = [...new Set(repositories.map((r) => r.provider_type))];

  if (loading) {
    return <CenteredSpinner />;
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">{t("repositories.title")}</h1>
          <p className="text-muted-foreground">
            {t("repositories.subtitle")}
          </p>
        </div>
        <Button onClick={() => setShowImportModal(true)}>
          <svg
            className="w-4 h-4 mr-2"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4v16m8-8H4"
            />
          </svg>
          {t("repositories.import")}
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="p-4 border border-border rounded-lg bg-card">
          <p className="text-sm text-muted-foreground">{t("repositories.stats.total")}</p>
          <p className="text-2xl font-bold">{repositories.length}</p>
        </div>
        <div className="p-4 border border-border rounded-lg bg-card">
          <p className="text-sm text-muted-foreground">{t("repositories.stats.active")}</p>
          <p className="text-2xl font-bold">
            {repositories.filter((r) => r.is_active).length}
          </p>
        </div>
        <div className="p-4 border border-border rounded-lg bg-card">
          <p className="text-sm text-muted-foreground">{t("repositories.stats.providers")}</p>
          <p className="text-2xl font-bold">{providerTypes.length}</p>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex gap-4 mb-6">
        <div className="flex-1">
          <Input
            placeholder={t("repositories.search")}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="max-w-md"
          />
        </div>
        <select
          value={providerFilter}
          onChange={(e) => setProviderFilter(e.target.value)}
          className="px-3 py-2 border border-border rounded-md bg-background"
        >
          <option value="">{t("repositories.allProviders")}</option>
          {providerTypes.map((type) => (
            <option key={type} value={type}>
              {type}
            </option>
          ))}
        </select>
      </div>

      {/* Repository List */}
      <div className="border border-border rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-muted">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium">{t("repositories.columns.name")}</th>
              <th className="px-4 py-3 text-left text-sm font-medium">{t("repositories.columns.provider")}</th>
              <th className="px-4 py-3 text-left text-sm font-medium">{t("repositories.columns.branch")}</th>
              <th className="px-4 py-3 text-left text-sm font-medium">{t("repositories.columns.status")}</th>
              <th className="px-4 py-3 text-right text-sm font-medium">{t("repositories.columns.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {filteredRepositories.map((repo) => (
              <tr key={repo.id} className="hover:bg-muted/50">
                <td className="px-4 py-3">
                  <Link
                    href={`/${orgSlug}/repositories/${repo.id}`}
                    className="font-medium hover:text-primary"
                  >
                    {repo.name}
                  </Link>
                  <p className="text-sm text-muted-foreground">{repo.full_path}</p>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <GitProviderIcon provider={repo.provider_type} className="w-4 h-4" />
                    <span className="capitalize">{repo.provider_type}</span>
                  </div>
                </td>
                <td className="px-4 py-3 text-muted-foreground">
                  {repo.default_branch}
                </td>
                <td className="px-4 py-3">
                  <span
                    className={`inline-flex px-2 py-0.5 text-xs rounded ${
                      repo.is_active
                        ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                        : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
                    }`}
                  >
                    {repo.is_active ? t("repositories.active") : t("repositories.inactive")}
                  </span>
                </td>
                <td className="px-4 py-3 text-right">
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => handleDelete(repo.id)}
                  >
                    {t("repositories.delete")}
                  </Button>
                </td>
              </tr>
            ))}
            {filteredRepositories.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                  {repositories.length === 0
                    ? t("repositories.emptyState.title")
                    : t("repositories.emptyState.noMatch")}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Import Modal */}
      <ImportRepositoryModal
        open={showImportModal}
        onClose={() => setShowImportModal(false)}
        onImported={() => {
          setShowImportModal(false);
          loadData();
        }}
        existingRepositories={repositories}
      />

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog {...deleteDialog.dialogProps} />
    </div>
  );
}
