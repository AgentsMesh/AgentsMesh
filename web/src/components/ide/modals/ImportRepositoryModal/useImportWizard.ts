"use client";

import { useState, useCallback } from "react";
import {
  repositoryApi,
  userRepositoryProviderApi,
  RepositoryProviderData,
  UserRemoteRepositoryData,
  RepositoryData,
} from "@/lib/api";
import type { ImportWizardState, ImportWizardActions, ImportWizardStep } from "./types";

/**
 * Creates the initial state for the import wizard.
 * Call this function to get a fresh state object.
 */
function createInitialState(): ImportWizardState {
  return {
    step: "source",
    providers: [],
    selectedProvider: null,
    repositories: [],
    selectedRepo: null,
    search: "",
    page: 1,
    loadingProviders: false,
    loadingRepos: false,
    importing: false,
    error: null,
    manualProviderType: "github",
    manualBaseURL: "https://github.com",
    manualCloneURL: "",
    manualName: "",
    manualFullPath: "",
    manualDefaultBranch: "main",
    ticketPrefix: "",
    visibility: "organization",
  };
}

interface UseImportWizardOptions {
  onClose: () => void;
  onImported?: () => void;
  existingRepositories?: RepositoryData[];
  t: (key: string) => string;
  /**
   * Callback invoked once when the hook is first used.
   * The parent component should call this to trigger initial data loading.
   */
  onInit?: (actions: Pick<ImportWizardActions, "loadProviders">) => void;
}

/**
 * Hook for managing import repository wizard state and actions.
 *
 * This hook follows React best practices by avoiding useEffect for data fetching.
 * Instead, data loading is triggered explicitly via:
 * 1. Parent component calling actions.loadProviders() on mount
 * 2. selectProvider action triggering repository loading
 *
 * State reset is handled by the parent component using the key pattern.
 */
export function useImportWizard({
  onClose,
  onImported,
  existingRepositories: _existingRepositories = [],
  t,
}: UseImportWizardOptions): [ImportWizardState, ImportWizardActions] {
  // Note: existingRepositories is available for future duplicate detection
  void _existingRepositories;
  const [state, setState] = useState<ImportWizardState>(createInitialState);

  // Load providers - call this explicitly, not via useEffect
  const loadProviders = useCallback(async () => {
    try {
      setState(s => ({ ...s, loadingProviders: true }));
      const response = await userRepositoryProviderApi.list();
      const activeProviders = (response.providers || []).filter(
        (p) => p.is_active && (p.has_identity || p.has_bot_token)
      );
      setState(s => ({ ...s, providers: activeProviders, loadingProviders: false }));
    } catch (err) {
      console.error("Failed to load providers:", err);
      setState(s => ({
        ...s,
        error: t("repositories.modal.failedToLoadConnections"),
        loadingProviders: false,
      }));
    }
  }, [t]);

  // Load repositories for selected provider
  const loadRepositories = useCallback(async () => {
    if (!state.selectedProvider) return;
    try {
      setState(s => ({ ...s, loadingRepos: true, error: null }));
      const response = await userRepositoryProviderApi.listRepositories(state.selectedProvider.id, {
        page: state.page,
        perPage: 20,
        search: state.search || undefined,
      });
      setState(s => ({
        ...s,
        repositories: response.repositories || [],
        loadingRepos: false,
      }));
    } catch (err) {
      console.error("Failed to load repositories:", err);
      setState(s => ({
        ...s,
        error: t("repositories.modal.failedToLoadRepos"),
        loadingRepos: false,
      }));
    }
  }, [state.selectedProvider, state.page, state.search, t]);

  // Select provider and immediately load repositories (event-driven, not effect-driven)
  const selectProvider = useCallback((provider: RepositoryProviderData) => {
    setState(s => ({ ...s, selectedProvider: provider, step: "browse", loadingRepos: true }));

    // Directly trigger repository loading (not via useEffect)
    userRepositoryProviderApi.listRepositories(provider.id, {
      page: 1,
      perPage: 20,
      search: undefined,
    }).then(response => {
      setState(s => ({
        ...s,
        repositories: response.repositories || [],
        loadingRepos: false,
      }));
    }).catch(err => {
      console.error("Failed to load repositories:", err);
      setState(s => ({
        ...s,
        error: t("repositories.modal.failedToLoadRepos"),
        loadingRepos: false,
      }));
    });
  }, [t]);

  const actions: ImportWizardActions = {
    setStep: (step: ImportWizardStep) => setState(s => ({ ...s, step })),
    setSearch: (search: string) => setState(s => ({ ...s, search })),
    setPage: (page) => setState(s => ({
      ...s,
      page: typeof page === "function" ? page(s.page) : page,
    })),
    setError: (error) => setState(s => ({ ...s, error })),

    selectProvider,

    clearProvider: () => {
      setState(s => ({
        ...s,
        selectedProvider: null,
        repositories: [],
        step: "source",
      }));
    },

    selectRepo: (repo: UserRemoteRepositoryData, existingRepos: RepositoryData[]) => {
      const existingRepo = existingRepos.find(
        (r) => r.clone_url === repo.clone_url || r.full_path === repo.full_path
      );
      setState(s => ({
        ...s,
        selectedRepo: repo,
        manualName: repo.name,
        manualFullPath: repo.full_path,
        manualDefaultBranch: repo.default_branch || "main",
        manualCloneURL: repo.clone_url,
        manualProviderType: s.selectedProvider?.provider_type || "github",
        manualBaseURL: s.selectedProvider?.base_url || "https://github.com",
        ticketPrefix: existingRepo?.ticket_prefix || "",
        step: "confirm",
      }));
    },

    setManualProviderType: (type: string) => {
      let baseURL = "";
      switch (type) {
        case "github":
          baseURL = "https://github.com";
          break;
        case "gitlab":
          baseURL = "https://gitlab.com";
          break;
        case "gitee":
          baseURL = "https://gitee.com";
          break;
        default:
          baseURL = "";
      }
      setState(s => ({ ...s, manualProviderType: type, manualBaseURL: baseURL }));
    },
    setManualBaseURL: (url: string) => setState(s => ({ ...s, manualBaseURL: url })),
    setManualCloneURL: (url: string) => setState(s => ({ ...s, manualCloneURL: url })),
    setManualName: (name: string) => setState(s => ({ ...s, manualName: name })),
    setManualFullPath: (path: string) => setState(s => ({ ...s, manualFullPath: path })),
    setManualDefaultBranch: (branch: string) => setState(s => ({ ...s, manualDefaultBranch: branch })),

    setTicketPrefix: (prefix: string) => setState(s => ({ ...s, ticketPrefix: prefix.toUpperCase() })),
    setVisibility: (visibility: string) => setState(s => ({ ...s, visibility })),

    loadProviders,
    loadRepositories,

    handleManualContinue: () => {
      if (!state.manualCloneURL || !state.manualName || !state.manualFullPath) {
        setState(s => ({ ...s, error: t("repositories.modal.fillRequiredFields") }));
        return false;
      }
      setState(s => ({ ...s, step: "confirm" }));
      return true;
    },

    handleImport: async () => {
      setState(s => ({ ...s, importing: true, error: null }));
      try {
        // When importing from a provider, pass both HTTP and SSH clone URLs if available
        const httpCloneUrl = state.selectedRepo?.clone_url || undefined;
        const sshCloneUrl = state.selectedRepo?.ssh_clone_url || undefined;

        await repositoryApi.create({
          provider_type: state.manualProviderType,
          provider_base_url: state.manualBaseURL,
          clone_url: state.manualCloneURL,
          http_clone_url: httpCloneUrl,
          ssh_clone_url: sshCloneUrl,
          external_id: state.selectedRepo?.id || state.manualFullPath.replace(/[^a-zA-Z0-9]/g, "-"),
          name: state.manualName,
          full_path: state.manualFullPath,
          default_branch: state.manualDefaultBranch || "main",
          ticket_prefix: state.ticketPrefix || undefined,
          visibility: state.visibility,
        });
        onImported?.();
        onClose();
      } catch (err) {
        console.error("Failed to import repository:", err);
        setState(s => ({
          ...s,
          error: t("repositories.modal.failedToImport"),
          importing: false,
        }));
      }
    },

    goBack: () => {
      setState(s => {
        if (s.step === "browse") {
          return { ...s, step: "source", selectedProvider: null, repositories: [] };
        }
        if (s.step === "manual") {
          return { ...s, step: "source" };
        }
        if (s.step === "confirm") {
          return { ...s, step: s.selectedRepo ? "browse" : "manual" };
        }
        return s;
      });
    },

    reset: () => setState(createInitialState),
  };

  return [state, actions];
}

export default useImportWizard;
