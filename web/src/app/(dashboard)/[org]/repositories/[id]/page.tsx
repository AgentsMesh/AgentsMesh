"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { repositoryApi, RepositoryData } from "@/lib/api";
import { useTranslations } from "@/lib/i18n/client";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";
import { EditRepositoryModal } from "./components";

export default function RepositoryDetailPage() {
  const t = useTranslations();
  const params = useParams();
  const router = useRouter();
  const repositoryId = Number(params.id);

  const [repository, setRepository] = useState<RepositoryData | null>(null);
  const [branches, setBranches] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingBranches, setLoadingBranches] = useState(false);
  const [activeTab, setActiveTab] = useState<"info" | "branches">("info");
  const [showEditModal, setShowEditModal] = useState(false);

  const loadRepository = useCallback(async () => {
    try {
      const res = await repositoryApi.get(repositoryId);
      setRepository(res.repository);
    } catch (error) {
      console.error("Failed to load repository:", error);
    } finally {
      setLoading(false);
    }
  }, [repositoryId]);

  useEffect(() => {
    loadRepository();
  }, [loadRepository]);

  const loadBranches = useCallback(async () => {
    if (!repository) return;
    setLoadingBranches(true);
    try {
      // Note: Branch listing requires access token which should come from user's Git connection
      // For now, this will show empty or require manual token input
      // const res = await repositoryApi.listBranches(repositoryId, accessToken);
      // setBranches(res.branches || []);
      setBranches([]);
    } catch (error) {
      console.error("Failed to load branches:", error);
    } finally {
      setLoadingBranches(false);
    }
  }, [repository]);

  const handleDelete = useCallback(async () => {
    if (!repository) return;
    if (
      !confirm(t("repositories.detail.confirmDelete", { name: repository.name }))
    ) {
      return;
    }
    try {
      await repositoryApi.delete(repositoryId);
      router.push("../repositories");
    } catch (error) {
      console.error("Failed to delete repository:", error);
    }
  }, [repository, repositoryId, router, t]);

  const handleSetupWebhook = useCallback(async () => {
    if (!repository) return;
    try {
      const res = await repositoryApi.setupWebhook(repositoryId);
      alert(res.message + (res.webhook_url ? `\n\nWebhook URL: ${res.webhook_url}` : ""));
    } catch (error) {
      console.error("Failed to setup webhook:", error);
      alert(t("repositories.detail.webhookFailed"));
    }
  }, [repository, repositoryId, t]);

  useEffect(() => {
    if (activeTab === "branches" && branches.length === 0 && repository) {
      loadBranches();
    }
  }, [activeTab, branches.length, repository, loadBranches]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!repository) {
    return (
      <div className="p-6">
        <div className="text-center py-12">
          <p className="text-muted-foreground mb-4">{t("repositories.detail.notFound")}</p>
          <Link href="../repositories">
            <Button variant="outline">{t("repositories.detail.backToList")}</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
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
            <p className="text-muted-foreground">{repository.full_path}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setShowEditModal(true)}>
            {t("common.edit")}
          </Button>
          <Button variant="destructive" onClick={handleDelete}>
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

      {/* Tabs */}
      <div className="border-b border-border mb-6">
        <div className="flex gap-4">
          <button
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "info"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("info")}
          >
            {t("repositories.detail.information")}
          </button>
          <button
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "branches"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("branches")}
          >
            {t("repositories.detail.branches")}
          </button>
        </div>
      </div>

      {/* Tab Content */}
      {activeTab === "info" && (
        <div className="grid gap-6 md:grid-cols-2">
          {/* Repository Info */}
          <div className="border border-border rounded-lg p-6">
            <h3 className="font-semibold mb-4">{t("repositories.detail.repoDetails")}</h3>
            <dl className="space-y-3">
              <div>
                <dt className="text-sm text-muted-foreground">{t("repositories.detail.name")}</dt>
                <dd className="font-medium">{repository.name}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">{t("repositories.detail.fullPath")}</dt>
                <dd className="font-medium">{repository.full_path}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">{t("repositories.detail.cloneUrl")}</dt>
                <dd className="font-medium text-sm break-all">{repository.clone_url}</dd>
              </div>
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

          {/* Git Provider Info (from self-contained fields) */}
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

          {/* Actions */}
          <div className="border border-border rounded-lg p-6 md:col-span-2">
            <h3 className="font-semibold mb-4">{t("repositories.detail.actions")}</h3>
            <div className="flex flex-wrap gap-3">
              <Button variant="outline" onClick={handleSetupWebhook}>
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
                    d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
                  />
                </svg>
                {t("repositories.detail.setupWebhook")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {activeTab === "branches" && (
        <div className="border border-border rounded-lg">
          <div className="p-4 border-b border-border flex items-center justify-between">
            <h3 className="font-semibold">{t("repositories.detail.branches")}</h3>
          </div>
          <div className="divide-y divide-border">
            {loadingBranches ? (
              <div className="p-8 text-center">
                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto"></div>
              </div>
            ) : branches.length > 0 ? (
              branches.map((branch) => (
                <div
                  key={branch}
                  className="px-4 py-3 flex items-center justify-between hover:bg-muted/50"
                >
                  <div className="flex items-center gap-2">
                    <svg
                      className="w-4 h-4 text-muted-foreground"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                      />
                    </svg>
                    <span className="font-medium">{branch}</span>
                    {branch === repository.default_branch && (
                      <span className="px-2 py-0.5 text-xs bg-primary/10 text-primary rounded">
                        {t("repositories.repository.default")}
                      </span>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <div className="p-8 text-center text-muted-foreground">
                <p className="mb-2">{t("repositories.detail.branchesRequireCredentials")}</p>
                <p className="text-sm">
                  {t("repositories.detail.configureGitConnection")}
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {showEditModal && (
        <EditRepositoryModal
          repository={repository}
          onClose={() => setShowEditModal(false)}
          onUpdated={() => {
            setShowEditModal(false);
            loadRepository();
          }}
        />
      )}
    </div>
  );
}
