"use client";

import { useRouter } from "next/navigation";
import { useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { CenteredSpinner } from "@/components/ui/spinner";
import { FolderGit2, Plus } from "lucide-react";
import { useRepositories, useRepositoryStore } from "@/stores/repository";
import { useCurrentOrg } from "@/stores/auth";
import { useAutoSelectFirst } from "@/hooks/useAutoSelectFirst";
import { useCtaModal } from "@/hooks/useCtaModal";
import { InfraRepositoryDetail } from "@/components/infra/InfraRepositoryDetail";
import { ImportRepositoryModal } from "@/components/ide/modals/ImportRepositoryModal";

export function RepoSection({
  orgSlug,
  selectedId,
  idMissing,
  onBack,
}: {
  orgSlug: string;
  selectedId: number;
  idMissing: boolean;
  onBack: () => void;
}) {
  const router = useRouter();
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const repositories = useRepositories();
  const loading = useRepositoryStore((s) => s.isLoading);
  const fetched = useRepositoryStore((s) => s.fetched);
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);
  const importModal = useCtaModal(fetchRepositories);

  useEffect(() => {
    if (currentOrg) fetchRepositories();
  }, [currentOrg, fetchRepositories]);

  const firstId = repositories[0]?.id ?? null;

  useAutoSelectFirst({
    firstId,
    idMissing,
    loading,
    fetched,
    onNavigate: useCallback(
      (id) => router.replace(`/${orgSlug}/infra?tab=repositories&id=${id}`),
      [router, orgSlug],
    ),
  });

  let body: React.ReactNode;
  if (loading && repositories.length === 0) {
    body = <CenteredSpinner className="h-64" />;
  } else if (idMissing && firstId == null) {
    body = (
      <EmptyState
        size="full"
        icon={<FolderGit2 className="h-12 w-12" />}
        title={t("repositories.emptyState.title")}
        description={t("repositories.emptyState.description")}
        actions={
          <Button onClick={importModal.open}>
            <Plus className="mr-1 h-4 w-4" />
            {t("repositories.import")}
          </Button>
        }
      />
    );
  } else if (Number.isNaN(selectedId)) {
    body = null;
  } else {
    body = <InfraRepositoryDetail repositoryId={selectedId} onBack={onBack} />;
  }

  return (
    <>
      {body}
      <ImportRepositoryModal
        open={importModal.isOpen}
        onClose={importModal.close}
        onImported={importModal.commit}
      />
    </>
  );
}
