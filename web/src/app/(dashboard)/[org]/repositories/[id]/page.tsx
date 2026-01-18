"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { repositoryApi, RepositoryData } from "@/lib/api";
import { useTranslations } from "@/lib/i18n/client";

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
  }, [repository, repositoryId, router]);

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

  const getProviderIcon = (providerType?: string) => {
    switch (providerType) {
      case "github":
        return (
          <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
          </svg>
        );
      case "gitlab":
        return (
          <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
            <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z" />
          </svg>
        );
      case "gitee":
        return (
          <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
            <path d="M11.984 0A12 12 0 000 12a12 12 0 0012 12 12 12 0 0012-12A12 12 0 0012 0a12 12 0 00-.016 0zm6.09 5.333c.328 0 .593.266.592.593v1.482a.594.594 0 01-.593.592H9.777c-.982 0-1.778.796-1.778 1.778v5.63c0 .327.266.592.593.592h5.63c.982 0 1.778-.796 1.778-1.778v-.296a.593.593 0 00-.592-.593h-4.15a.592.592 0 01-.592-.592v-1.482a.593.593 0 01.593-.592h6.815c.327 0 .593.265.593.592v3.408a4 4 0 01-4 4H5.926a.593.593 0 01-.593-.593V9.778a4.444 4.444 0 014.445-4.444h8.296z" />
          </svg>
        );
      default:
        return (
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
        );
    }
  };

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
            {getProviderIcon(repository.provider_type)}
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

function EditRepositoryModal({
  repository,
  onClose,
  onUpdated,
}: {
  repository: RepositoryData;
  onClose: () => void;
  onUpdated: () => void;
}) {
  const t = useTranslations();
  const [name, setName] = useState(repository.name);
  const [defaultBranch, setDefaultBranch] = useState(repository.default_branch);
  const [ticketPrefix, setTicketPrefix] = useState(repository.ticket_prefix || "");
  const [isActive, setIsActive] = useState(repository.is_active);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleUpdate = async () => {
    if (!name) {
      setError(t("repositories.edit.nameRequired"));
      return;
    }

    setLoading(true);
    setError("");

    try {
      await repositoryApi.update(repository.id, {
        name,
        default_branch: defaultBranch,
        ticket_prefix: ticketPrefix || undefined,
        is_active: isActive,
      });
      onUpdated();
    } catch (err) {
      console.error("Failed to update repository:", err);
      setError(t("repositories.edit.updateFailed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-6">
        <h2 className="text-xl font-semibold mb-4">{t("repositories.edit.title")}</h2>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive text-sm rounded-md">
            {error}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("repositories.edit.name")} <span className="text-destructive">*</span>
            </label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">{t("repositories.edit.defaultBranch")}</label>
            <Input
              value={defaultBranch}
              onChange={(e) => setDefaultBranch(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("repositories.edit.ticketPrefixOptional")}
            </label>
            <Input
              placeholder="PROJ"
              value={ticketPrefix}
              onChange={(e) => setTicketPrefix(e.target.value.toUpperCase())}
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="is-active"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="rounded border-border"
            />
            <label htmlFor="is-active" className="text-sm font-medium">
              {t("repositories.edit.active")}
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleUpdate} disabled={!name || loading}>
            {loading ? t("repositories.edit.saving") : t("repositories.edit.saveChanges")}
          </Button>
        </div>
      </div>
    </div>
  );
}
