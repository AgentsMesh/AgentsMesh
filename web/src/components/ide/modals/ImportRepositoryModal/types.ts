import type { RepositoryProviderData, UserRemoteRepositoryData, RepositoryData } from "@/lib/api";

/**
 * Wizard step types for repository import flow
 */
export type ImportWizardStep = "source" | "browse" | "manual" | "confirm";

/**
 * State for the import wizard
 */
export interface ImportWizardState {
  step: ImportWizardStep;
  providers: RepositoryProviderData[];
  selectedProvider: RepositoryProviderData | null;
  repositories: UserRemoteRepositoryData[];
  selectedRepo: UserRemoteRepositoryData | null;
  search: string;
  page: number;
  loadingProviders: boolean;
  loadingRepos: boolean;
  importing: boolean;
  error: string | null;

  // Manual input fields
  manualProviderType: string;
  manualBaseURL: string;
  manualCloneURL: string;
  manualName: string;
  manualSlug: string;
  manualDefaultBranch: string;

  // Confirmation fields
  ticketPrefix: string;
  visibility: string;
}

/**
 * Actions for the import wizard
 */
export interface ImportWizardActions {
  setStep: (step: ImportWizardStep) => void;
  setSearch: (search: string) => void;
  setPage: (page: number | ((p: number) => number)) => void;
  setError: (error: string | null) => void;

  // Provider actions
  selectProvider: (provider: RepositoryProviderData) => void;
  clearProvider: () => void;

  // Repository actions
  selectRepo: (repo: UserRemoteRepositoryData, existingRepositories: RepositoryData[]) => void;

  // Manual entry actions
  setManualProviderType: (type: string) => void;
  setManualBaseURL: (url: string) => void;
  setManualCloneURL: (url: string) => void;
  setManualName: (name: string) => void;
  setManualSlug: (slug: string) => void;
  setManualDefaultBranch: (branch: string) => void;

  // Confirmation actions
  setTicketPrefix: (prefix: string) => void;
  setVisibility: (visibility: string) => void;

  // Data loading
  loadProviders: () => Promise<void>;
  loadRepositories: () => Promise<void>;

  // Final import
  handleImport: () => Promise<void>;

  // Navigation
  handleManualContinue: () => boolean;
  goBack: () => void;
  reset: () => void;
}

/**
 * Props for ImportRepositoryModal
 */
export interface ImportRepositoryModalProps {
  open: boolean;
  onClose: () => void;
  onImported?: () => void;
  existingRepositories?: RepositoryData[];
}

/**
 * Props for step components
 */
export interface StepProps {
  state: ImportWizardState;
  actions: ImportWizardActions;
  existingRepositories?: RepositoryData[];
  t: (key: string, params?: Record<string, string | number>) => string;
}
