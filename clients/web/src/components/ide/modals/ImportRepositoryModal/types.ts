import type { RepositoryProviderData, RepositoryData } from "@/lib/api";
import type { ProviderRepositoryData } from "@/lib/api/userRepositoryProviderTypes";

export type ImportWizardStep = "source" | "browse" | "manual" | "confirm";

export interface ImportWizardState {
  step: ImportWizardStep;
  providers: RepositoryProviderData[];
  selectedProvider: RepositoryProviderData | null;
  repositories: ProviderRepositoryData[];
  selectedRepo: ProviderRepositoryData | null;
  search: string;
  page: number;
  loadingProviders: boolean;
  loadingRepos: boolean;
  importing: boolean;
  error: string | null;

  manualProviderType: string;
  manualBaseURL: string;
  manualCloneURL: string;
  manualName: string;
  manualSlug: string;
  manualDefaultBranch: string;

  ticketPrefix: string;
  visibility: string;
}

export interface ImportWizardActions {
  setStep: (step: ImportWizardStep) => void;
  setSearch: (search: string) => void;
  setPage: (page: number | ((p: number) => number)) => void;
  setError: (error: string | null) => void;

  selectProvider: (provider: RepositoryProviderData) => void;
  clearProvider: () => void;

  selectRepo: (repo: ProviderRepositoryData, existingRepositories: RepositoryData[]) => void;

  setManualProviderType: (type: string) => void;
  setManualBaseURL: (url: string) => void;
  setManualCloneURL: (url: string) => void;
  setManualName: (name: string) => void;
  setManualSlug: (slug: string) => void;
  setManualDefaultBranch: (branch: string) => void;

  setTicketPrefix: (prefix: string) => void;
  setVisibility: (visibility: string) => void;

  loadProviders: () => Promise<void>;
  loadRepositories: () => Promise<void>;

  handleImport: () => Promise<void>;

  handleManualContinue: () => boolean;
  goBack: () => void;
  reset: () => void;
}

export interface ImportRepositoryModalProps {
  open: boolean;
  onClose: () => void;
  onImported?: () => void;
  existingRepositories?: RepositoryData[];
}

export interface StepProps {
  state: ImportWizardState;
  actions: ImportWizardActions;
  existingRepositories?: RepositoryData[];
  t: (key: string, params?: Record<string, string | number>) => string;
}
