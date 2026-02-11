"use client";

import { useTranslations } from "next-intl";

interface BranchesTabProps {
  branches: string[];
  defaultBranch: string;
  loading: boolean;
}

export function BranchesTab({ branches, defaultBranch, loading }: BranchesTabProps) {
  const t = useTranslations();

  return (
    <div className="border border-border rounded-lg">
      <div className="p-4 border-b border-border flex items-center justify-between">
        <h3 className="font-semibold">{t("repositories.detail.branches")}</h3>
      </div>
      <div className="divide-y divide-border">
        {loading ? (
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
                {branch === defaultBranch && (
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
  );
}
