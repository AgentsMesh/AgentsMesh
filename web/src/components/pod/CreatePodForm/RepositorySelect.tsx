"use client";

import type { RepositoryData } from "@/lib/api";

interface RepositorySelectProps {
  repositories: RepositoryData[];
  selectedRepositoryId: number | null;
  onSelect: (repositoryId: number | null) => void;
  t: (key: string) => string;
}

/**
 * Repository selection dropdown component
 */
export function RepositorySelect({
  repositories,
  selectedRepositoryId,
  onSelect,
  t,
}: RepositorySelectProps) {
  return (
    <div>
      <label
        htmlFor="repository-select"
        className="block text-sm font-medium mb-2"
      >
        {t("ide.createPod.selectRepository")}
      </label>
      <select
        id="repository-select"
        className="w-full px-3 py-2 border border-border rounded-md bg-background"
        value={selectedRepositoryId || ""}
        onChange={(e) =>
          onSelect(e.target.value ? Number(e.target.value) : null)
        }
      >
        <option value="">{t("ide.createPod.selectRepositoryPlaceholder")}</option>
        {repositories.map((repo) => (
          <option key={repo.id} value={repo.id}>
            {repo.slug}
          </option>
        ))}
      </select>
    </div>
  );
}

interface BranchInputProps {
  value: string;
  onChange: (value: string) => void;
  error?: string;
  t: (key: string) => string;
}

/**
 * Branch input component
 */
export function BranchInput({
  value,
  onChange,
  error,
  t,
}: BranchInputProps) {
  return (
    <div>
      <label
        htmlFor="branch-input"
        className="block text-sm font-medium mb-2"
      >
        {t("ide.createPod.branch")}
      </label>
      <input
        id="branch-input"
        type="text"
        className={`w-full px-3 py-2 border rounded-md bg-background ${
          error ? "border-destructive" : "border-border"
        }`}
        placeholder={t("ide.createPod.branchPlaceholder")}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={!!error}
        aria-describedby={error ? "branch-error" : undefined}
      />
      {error && (
        <p id="branch-error" className="text-xs text-destructive mt-1">
          {error}
        </p>
      )}
    </div>
  );
}
