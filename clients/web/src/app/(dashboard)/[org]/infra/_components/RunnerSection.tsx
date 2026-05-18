"use client";

import { useRouter } from "next/navigation";
import { useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { CenteredSpinner } from "@/components/ui/spinner";
import { Server, Plus } from "lucide-react";
import { useRunners, useRunnerStore } from "@/stores/runner";
import { useCurrentOrg } from "@/stores/auth";
import { useAutoSelectFirst } from "@/hooks/useAutoSelectFirst";
import { useCtaModal } from "@/hooks/useCtaModal";
import { InfraRunnerDetail } from "@/components/infra/InfraRunnerDetail";
import { AddRunnerModal } from "@/components/ide/modals/AddRunnerModal";

export function RunnerSection({
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
  const runners = useRunners();
  const loading = useRunnerStore((s) => s.loading);
  const fetched = useRunnerStore((s) => s.fetched);
  const fetchRunners = useRunnerStore((s) => s.fetchRunners);
  const addModal = useCtaModal(fetchRunners);

  useEffect(() => {
    if (currentOrg) fetchRunners();
  }, [currentOrg, fetchRunners]);

  const firstId = runners[0]?.id ?? null;

  useAutoSelectFirst({
    firstId,
    idMissing,
    loading,
    fetched,
    onNavigate: useCallback(
      (id) => router.replace(`/${orgSlug}/infra?tab=runners&id=${id}`),
      [router, orgSlug],
    ),
  });

  let body: React.ReactNode;
  if (loading && runners.length === 0) {
    body = <CenteredSpinner className="h-64" />;
  } else if (idMissing && firstId == null) {
    body = (
      <EmptyState
        size="full"
        icon={<Server className="h-12 w-12" />}
        title={t("runners.emptyState.title")}
        description={t("runners.emptyState.description")}
        actions={
          <Button onClick={addModal.open}>
            <Plus className="mr-1 h-4 w-4" />
            {t("runners.addRunner")}
          </Button>
        }
      />
    );
  } else if (Number.isNaN(selectedId)) {
    body = null;
  } else {
    body = <InfraRunnerDetail runnerId={selectedId} onBack={onBack} />;
  }

  return (
    <>
      {body}
      <AddRunnerModal
        open={addModal.isOpen}
        onClose={addModal.close}
        onCreated={addModal.commit}
      />
    </>
  );
}
