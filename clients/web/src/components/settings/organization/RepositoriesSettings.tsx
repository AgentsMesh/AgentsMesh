"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { useRepositories, useRepositoryStore, type Repository } from "@/stores/repository";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";
import { ImportRepositoryModal } from "@/components/ide/modals/ImportRepositoryModal/index";
import { FolderGit2, Trash2, Plus } from "lucide-react";

/** Organization > Infrastructure > Repositories tab. */
export function RepositoriesSettings() {
  const { org: orgSlug } = useParams<{ org: string }>();
  const t = useTranslations();
  const repositories = useRepositories();
  const loading = useRepositoryStore((s) => s.isLoading);
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);
  const deleteRepository = useRepositoryStore((s) => s.deleteRepository);
  const [filter, setFilter] = useState("");
  const [showImportModal, setShowImportModal] = useState(false);

  const { confirm: confirmDelete, dialogProps: deleteDialogProps } = useConfirmDialog({
    title: t("repositories.deleteDialog.title"),
    description: t("repositories.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  useEffect(() => {
    fetchRepositories();
  }, [fetchRepositories]);

  const handleDelete = async (repo: Repository) => {
    const ok = await confirmDelete();
    if (!ok) return;
    try {
      await deleteRepository(repo.id);
      toast.success(t("repositories.deleteSuccess"));
    } catch (err) {
      toast.error(getLocalizedErrorMessage(err, t, t("common.error")));
    }
  };

  const filtered = repositories.filter((r) =>
    !filter || r.slug?.toLowerCase().includes(filter.toLowerCase()),
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-xl font-semibold tracking-tight">
          {t("infra.tabs.repositories")}
        </h2>
        <Button onClick={() => setShowImportModal(true)}>
          <Plus className="w-4 h-4 mr-1" />
          {t("repositories.import")}
        </Button>
      </div>

      <Input
        placeholder={t("repositories.searchPlaceholder")}
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />

      {loading ? (
        <CenteredSpinner />
      ) : filtered.length === 0 ? (
        <EmptyState
          size="default"
          icon={<FolderGit2 className="w-10 h-10" />}
          title={t("repositories.emptyState.title")}
          description={t("repositories.emptyState.description")}
          actions={
            <Button onClick={() => setShowImportModal(true)}>
              <Plus className="w-4 h-4 mr-1" />
              {t("repositories.import")}
            </Button>
          }
        />
      ) : (
        <ul className="divide-y divide-border border border-border rounded-md">
          {filtered.map((repo) => (
            <li key={repo.id} className="flex items-center justify-between gap-3 px-4 py-3">
              <div className="flex items-center gap-3 min-w-0">
                <GitProviderIcon
                  provider={repo.provider_type ?? undefined}
                  className="w-5 h-5 text-muted-foreground flex-shrink-0"
                />
                <div className="min-w-0">
                  <Link
                    href={`/${orgSlug}/infra?tab=repositories&id=${repo.id}`}
                    className="font-medium text-sm truncate hover:underline"
                  >
                    {repo.slug}
                  </Link>
                  {repo.default_branch && (
                    <div className="text-xs text-muted-foreground">{repo.default_branch}</div>
                  )}
                </div>
              </div>
              <Button variant="ghost" size="icon" onClick={() => handleDelete(repo)}>
                <Trash2 className="w-4 h-4" />
              </Button>
            </li>
          ))}
        </ul>
      )}

      <ConfirmDialog {...deleteDialogProps} />
      <ImportRepositoryModal
        open={showImportModal}
        onClose={() => setShowImportModal(false)}
        onImported={fetchRepositories}
        existingRepositories={repositories}
      />
    </div>
  );
}
