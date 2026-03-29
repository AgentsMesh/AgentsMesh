"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CenteredSpinner } from "@/components/ui/spinner";
import type { StepProps } from "../types";

/**
 * Step 2: Browse Repositories - Search and select from provider
 */
export function BrowseStep({ state, actions, existingRepositories = [], t }: StepProps) {
  const { selectedProvider, repositories, search, page, loadingRepos } = state;

  if (!selectedProvider) return null;

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    actions.setPage(1);
    actions.loadRepositories();
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <button
          onClick={actions.goBack}
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

      <form onSubmit={handleSearch} className="flex gap-2">
        <Input
          value={search}
          onChange={(e) => actions.setSearch(e.target.value)}
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
              onClick={() => actions.selectRepo(repo, existingRepositories)}
              className="w-full flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50 text-left"
            >
              <div>
                <div className="font-medium">{repo.slug}</div>
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
            onClick={() => actions.setPage((p) => p - 1)}
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
            onClick={() => actions.setPage((p) => p + 1)}
          >
            {t("repositories.modal.next")}
          </Button>
        </div>
      )}
    </div>
  );
}

export default BrowseStep;
