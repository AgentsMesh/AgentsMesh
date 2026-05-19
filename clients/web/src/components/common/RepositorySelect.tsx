"use client";

import { useEffect, useMemo } from "react";
import { RepositoryData } from "@/lib/api";
import { useRepositories, useRepositoryStore } from "@/stores/repository";

export interface RepositorySelectProps {
  value: number | null;
  onChange: (value: number | null, repository?: RepositoryData) => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
  activeOnly?: boolean;
}

export function RepositorySelect({
  value,
  onChange,
  disabled = false,
  placeholder = "Select a repository...",
  className = "",
  activeOnly = true,
}: RepositorySelectProps) {
  const allRepos = useRepositories();
  const loading = useRepositoryStore((s) => s.isLoading);
  const error = useRepositoryStore((s) => s.error);
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);

  useEffect(() => { fetchRepositories(); }, [fetchRepositories]);

  const repositories = useMemo(
    () => (activeOnly ? allRepos.filter((r) => r.is_active) : allRepos),
    [allRepos, activeOnly],
  );

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedValue = e.target.value;
    if (!selectedValue) {
      onChange(null);
    } else {
      const selectedId = Number(selectedValue);
      const selectedRepo = repositories.find((r) => r.id === selectedId);
      onChange(selectedId, selectedRepo);
    }
  };

  if (error) {
    return (
      <div className={`text-sm text-destructive ${className}`}>
        {error}
        <button
          type="button"
          onClick={() => fetchRepositories()}
          className="ml-2 underline hover:no-underline"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <select
      className={`w-full px-3 py-2 border border-border rounded-md bg-background ${className}`}
      value={value || ""}
      onChange={handleChange}
      disabled={disabled || loading}
    >
      <option value="">
        {loading ? "Loading repositories..." : placeholder}
      </option>
      {repositories.map((repo) => (
        <option key={repo.id} value={repo.id}>
          {repo.slug}
        </option>
      ))}
    </select>
  );
}

export default RepositorySelect;
