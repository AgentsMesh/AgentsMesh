"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CenteredSpinner } from "@/components/ui/spinner";
import {
  repositoryApi,
  userRepositoryProviderApi,
  RepositoryProviderData,
  UserRemoteRepositoryData,
} from "@/lib/api";
import type { RepositoryData } from "@/lib/api";
import { useTranslations } from "@/lib/i18n/client";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";

interface ImportRepositoryModalProps {
  open: boolean;
  onClose: () => void;
  onImported?: () => void;
  existingRepositories?: RepositoryData[];
}

/**
 * ImportRepositoryModal - Modal for importing repositories from git providers
 *
 * Extracted from repositories/page.tsx to be reusable across the application
 */
export function ImportRepositoryModal({
  open,
  onClose,
  onImported,
  existingRepositories = [],
}: ImportRepositoryModalProps) {
  const t = useTranslations();
  const [step, setStep] = useState<"source" | "browse" | "manual" | "confirm">("source");
  const [providers, setProviders] = useState<RepositoryProviderData[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<RepositoryProviderData | null>(null);
  const [repositories, setRepositories] = useState<UserRemoteRepositoryData[]>([]);
  const [selectedRepo, setSelectedRepo] = useState<UserRemoteRepositoryData | null>(null);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [loadingProviders, setLoadingProviders] = useState(true);
  const [loadingRepos, setLoadingRepos] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Manual input fields
  const [manualProviderType, setManualProviderType] = useState("github");
  const [manualBaseURL, setManualBaseURL] = useState("https://github.com");
  const [manualCloneURL, setManualCloneURL] = useState("");
  const [manualName, setManualName] = useState("");
  const [manualFullPath, setManualFullPath] = useState("");
  const [manualDefaultBranch, setManualDefaultBranch] = useState("main");

  // Confirmation fields
  const [ticketPrefix, setTicketPrefix] = useState("");
  const [visibility, setVisibility] = useState("organization");

  const loadProviders = useCallback(async () => {
    try {
      setLoadingProviders(true);
      const response = await userRepositoryProviderApi.list();
      // Only show providers that have authentication configured
      const activeProviders = (response.providers || []).filter(
        (p) => p.is_active && (p.has_identity || p.has_bot_token)
      );
      setProviders(activeProviders);
    } catch (err) {
      console.error("Failed to load providers:", err);
      setError(t("repositories.modal.failedToLoadConnections"));
    } finally {
      setLoadingProviders(false);
    }
  }, [t]);

  useEffect(() => {
    if (open) {
      loadProviders();
    }
  }, [open, loadProviders]);

  const loadRepositories = useCallback(async () => {
    if (!selectedProvider) return;
    try {
      setLoadingRepos(true);
      setError(null);
      const response = await userRepositoryProviderApi.listRepositories(selectedProvider.id, {
        page,
        perPage: 20,
        search: search || undefined,
      });
      setRepositories(response.repositories || []);
    } catch (err) {
      console.error("Failed to load repositories:", err);
      setError(t("repositories.modal.failedToLoadRepos"));
    } finally {
      setLoadingRepos(false);
    }
  }, [selectedProvider, page, search, t]);

  useEffect(() => {
    if (step === "browse" && selectedProvider) {
      loadRepositories();
    }
  }, [step, selectedProvider, loadRepositories]);

  // Reset state when modal closes
  useEffect(() => {
    if (!open) {
      setStep("source");
      setSelectedProvider(null);
      setSelectedRepo(null);
      setRepositories([]);
      setSearch("");
      setPage(1);
      setError(null);
      setManualProviderType("github");
      setManualBaseURL("https://github.com");
      setManualCloneURL("");
      setManualName("");
      setManualFullPath("");
      setManualDefaultBranch("main");
      setTicketPrefix("");
      setVisibility("organization");
    }
  }, [open]);

  if (!open) return null;

  const handleSelectProvider = (provider: RepositoryProviderData) => {
    setSelectedProvider(provider);
    setStep("browse");
  };

  const handleSelectRepo = (repo: UserRemoteRepositoryData) => {
    setSelectedRepo(repo);
    setManualName(repo.name);
    setManualFullPath(repo.full_path);
    setManualDefaultBranch(repo.default_branch || "main");
    setManualCloneURL(repo.clone_url);
    if (selectedProvider) {
      setManualProviderType(selectedProvider.provider_type);
      setManualBaseURL(selectedProvider.base_url);
    }

    // Look up existing repository's ticket_prefix
    const existingRepo = existingRepositories.find(
      (r) => r.clone_url === repo.clone_url || r.full_path === repo.full_path
    );
    setTicketPrefix(existingRepo?.ticket_prefix || "");

    setStep("confirm");
  };

  const handleManualContinue = () => {
    if (!manualCloneURL || !manualName || !manualFullPath) {
      setError(t("repositories.modal.fillRequiredFields"));
      return;
    }
    setStep("confirm");
  };

  const handleImport = async () => {
    setImporting(true);
    setError(null);
    try {
      await repositoryApi.create({
        provider_type: manualProviderType,
        provider_base_url: manualBaseURL,
        clone_url: manualCloneURL,
        external_id: selectedRepo?.id || manualFullPath.replace(/[^a-zA-Z0-9]/g, "-"),
        name: manualName,
        full_path: manualFullPath,
        default_branch: manualDefaultBranch || "main",
        ticket_prefix: ticketPrefix || undefined,
        visibility: visibility,
      });
      onImported?.();
      onClose();
    } catch (err) {
      console.error("Failed to import repository:", err);
      setError(t("repositories.modal.failedToImport"));
    } finally {
      setImporting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-2xl mx-4 max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("repositories.modal.title")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <div className="flex-1 overflow-auto p-4">
          {error && (
            <div className="mb-4 p-3 bg-destructive/10 text-destructive text-sm rounded-lg">
              {error}
            </div>
          )}

          {/* Step 1: Select Source */}
          {step === "source" && (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                {t("repositories.modal.selectSourceHint")}
              </p>

              {loadingProviders ? (
                <CenteredSpinner className="py-8" />
              ) : (
                <>
                  <div className="space-y-2">
                    <p className="text-sm font-medium">{t("repositories.modal.yourConnections")}</p>
                    {providers.length === 0 ? (
                      <p className="text-sm text-muted-foreground py-4">
                        {t("repositories.modal.noConnections")}{" "}
                        <Link
                          href="/settings/repository-providers"
                          className="text-primary hover:underline"
                        >
                          {t("repositories.modal.addOne")}
                        </Link>{" "}
                        {t("repositories.modal.toBrowse")}
                      </p>
                    ) : (
                      <div className="grid grid-cols-2 gap-3">
                        {providers.map((provider) => (
                          <button
                            key={provider.id}
                            onClick={() => handleSelectProvider(provider)}
                            className="flex items-center gap-3 p-4 border border-border rounded-lg hover:bg-muted/50 text-left"
                          >
                            <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                              <GitProviderIcon provider={provider.provider_type} />
                            </div>
                            <div>
                              <div className="font-medium">{provider.name}</div>
                              <div className="text-xs text-muted-foreground">
                                {provider.base_url}
                              </div>
                              {provider.has_identity && (
                                <div className="text-xs text-green-600 dark:text-green-400">
                                  OAuth
                                </div>
                              )}
                            </div>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="relative">
                    <div className="absolute inset-0 flex items-center">
                      <div className="w-full border-t border-border"></div>
                    </div>
                    <div className="relative flex justify-center text-xs uppercase">
                      <span className="bg-background px-2 text-muted-foreground">
                        {t("repositories.modal.or")}
                      </span>
                    </div>
                  </div>

                  <button
                    onClick={() => setStep("manual")}
                    className="w-full flex items-center gap-3 p-4 border border-dashed border-border rounded-lg hover:bg-muted/50"
                  >
                    <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                      <svg
                        className="w-5 h-5"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                        />
                      </svg>
                    </div>
                    <div className="text-left">
                      <div className="font-medium">{t("repositories.modal.enterManually")}</div>
                      <div className="text-xs text-muted-foreground">
                        {t("repositories.modal.enterManuallyHint")}
                      </div>
                    </div>
                  </button>
                </>
              )}
            </div>
          )}

          {/* Step 2: Browse Repositories */}
          {step === "browse" && selectedProvider && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <button
                  onClick={() => {
                    setStep("source");
                    setSelectedProvider(null);
                    setRepositories([]);
                  }}
                  className="text-muted-foreground hover:text-foreground"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 19l-7-7 7-7"
                    />
                  </svg>
                </button>
                <span className="text-sm text-muted-foreground">{selectedProvider.name}</span>
              </div>

              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  setPage(1);
                  loadRepositories();
                }}
                className="flex gap-2"
              >
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={t("repositories.searchPlaceholder")}
                  className="flex-1"
                />
                <Button type="submit">{t("common.search")}</Button>
              </form>

              {loadingRepos ? (
                <CenteredSpinner className="py-8" />
              ) : repositories.length === 0 ? (
                <p className="text-center text-muted-foreground py-8">
                  {t("repositories.modal.noReposFound")}
                </p>
              ) : (
                <div className="space-y-2 max-h-[300px] overflow-auto">
                  {repositories.map((repo) => (
                    <button
                      key={repo.id}
                      onClick={() => handleSelectRepo(repo)}
                      className="w-full flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50 text-left"
                    >
                      <div>
                        <div className="font-medium">{repo.full_path}</div>
                        <div className="text-sm text-muted-foreground line-clamp-1">
                          {repo.description || t("repositories.modal.noDescription")}
                        </div>
                        <div className="flex items-center gap-2 mt-1">
                          <span className="px-2 py-0.5 text-xs bg-muted rounded">
                            {repo.visibility}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            {repo.default_branch}
                          </span>
                        </div>
                      </div>
                      <svg
                        className="w-5 h-5 text-muted-foreground"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M9 5l7 7-7 7"
                        />
                      </svg>
                    </button>
                  ))}
                </div>
              )}

              {repositories.length > 0 && (
                <div className="flex items-center justify-between pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage((p) => p - 1)}
                  >
                    {t("repositories.modal.previous")}
                  </Button>
                  <span className="text-sm text-muted-foreground">
                    {t("repositories.modal.page", { page })}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={repositories.length < 20}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    {t("repositories.modal.next")}
                  </Button>
                </div>
              )}
            </div>
          )}

          {/* Step 3: Manual Entry */}
          {step === "manual" && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setStep("source")}
                  className="text-muted-foreground hover:text-foreground"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 19l-7-7 7-7"
                    />
                  </svg>
                </button>
                <span className="text-sm text-muted-foreground">
                  {t("repositories.modal.manualEntry")}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    {t("repositories.modal.providerType")}
                  </label>
                  <select
                    className="w-full px-3 py-2 border border-border rounded-md bg-background"
                    value={manualProviderType}
                    onChange={(e) => {
                      setManualProviderType(e.target.value);
                      switch (e.target.value) {
                        case "github":
                          setManualBaseURL("https://github.com");
                          break;
                        case "gitlab":
                          setManualBaseURL("https://gitlab.com");
                          break;
                        case "gitee":
                          setManualBaseURL("https://gitee.com");
                          break;
                        default:
                          setManualBaseURL("");
                      }
                    }}
                  >
                    <option value="github">GitHub</option>
                    <option value="gitlab">GitLab</option>
                    <option value="gitee">Gitee</option>
                    <option value="generic">{t("repositories.modal.genericGit")}</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    {t("repositories.modal.baseUrl")}
                  </label>
                  <Input
                    value={manualBaseURL}
                    onChange={(e) => setManualBaseURL(e.target.value)}
                    placeholder="https://github.com"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("repositories.modal.cloneUrl")} *
                </label>
                <Input
                  value={manualCloneURL}
                  onChange={(e) => setManualCloneURL(e.target.value)}
                  placeholder="https://github.com/org/repo.git"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    {t("repositories.modal.repoName")} *
                  </label>
                  <Input
                    value={manualName}
                    onChange={(e) => setManualName(e.target.value)}
                    placeholder="my-project"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    {t("repositories.modal.fullPath")} *
                  </label>
                  <Input
                    value={manualFullPath}
                    onChange={(e) => setManualFullPath(e.target.value)}
                    placeholder="org/my-project"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("repositories.modal.defaultBranch")}
                </label>
                <Input
                  value={manualDefaultBranch}
                  onChange={(e) => setManualDefaultBranch(e.target.value)}
                  placeholder="main"
                />
              </div>
            </div>
          )}

          {/* Step 4: Confirm */}
          {step === "confirm" && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setStep(selectedRepo ? "browse" : "manual")}
                  className="text-muted-foreground hover:text-foreground"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 19l-7-7 7-7"
                    />
                  </svg>
                </button>
                <span className="text-sm text-muted-foreground">
                  {t("repositories.modal.confirmImport")}
                </span>
              </div>

              <div className="p-4 border border-border rounded-lg bg-muted/50">
                <div className="flex items-center gap-3 mb-3">
                  <GitProviderIcon provider={manualProviderType} />
                  <div>
                    <div className="font-medium">{manualName}</div>
                    <div className="text-sm text-muted-foreground">{manualFullPath}</div>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="text-muted-foreground">{t("repositories.modal.cloneUrl")}</div>
                  <div className="truncate">{manualCloneURL}</div>
                  <div className="text-muted-foreground">{t("repositories.modal.branch")}</div>
                  <div>{manualDefaultBranch}</div>
                  <div className="text-muted-foreground">{t("repositories.modal.provider")}</div>
                  <div className="capitalize">{manualProviderType}</div>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("repositories.modal.ticketPrefixOptional")}
                </label>
                <Input
                  value={ticketPrefix}
                  onChange={(e) => setTicketPrefix(e.target.value.toUpperCase())}
                  placeholder="PROJ"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  {t("repositories.modal.ticketPrefixHint")}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("repositories.modal.visibility")}
                </label>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2">
                    <input
                      type="radio"
                      checked={visibility === "organization"}
                      onChange={() => setVisibility("organization")}
                      className="w-4 h-4"
                    />
                    <span className="text-sm">{t("repositories.modal.organization")}</span>
                  </label>
                  <label className="flex items-center gap-2">
                    <input
                      type="radio"
                      checked={visibility === "private"}
                      onChange={() => setVisibility("private")}
                      className="w-4 h-4"
                    />
                    <span className="text-sm">{t("repositories.modal.privateOnly")}</span>
                  </label>
                </div>
              </div>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-3 p-4 border-t border-border">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          {step === "manual" && (
            <Button onClick={handleManualContinue}>{t("repositories.modal.continue")}</Button>
          )}
          {step === "confirm" && (
            <Button onClick={handleImport} disabled={importing}>
              {importing ? "..." : t("repositories.modal.importRepository")}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

export default ImportRepositoryModal;
