"use client";

import { useSearchParams, useRouter, useParams } from "next/navigation";
import { useEffect, useCallback, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { CenteredSpinner } from "@/components/ui/spinner";
import { FolderGit2, Server, Plus } from "lucide-react";
import { getRepositoryService } from "@/lib/wasm-core";
import type { RepositoryData } from "@/lib/api/repositoryTypes";
import { useRunners, useRunnerStore } from "@/stores/runner";
import { useCurrentOrg } from "@/stores/auth";
import { InfraRepositoryDetail } from "@/components/infra/InfraRepositoryDetail";
import { InfraRunnerDetail } from "@/components/infra/InfraRunnerDetail";
import { AddRunnerModal } from "@/components/ide/modals/AddRunnerModal";

type InfraTab = "repositories" | "runners";

export default function InfraPage() {
  const router = useRouter();
  const params = useParams<{ org: string }>();
  const searchParams = useSearchParams();
  const t = useTranslations();

  const tab = (searchParams.get("tab") as InfraTab) ?? "runners";
  const idParam = searchParams.get("id");
  const selectedId = idParam ? Number(idParam) : NaN;

  useEffect(() => {
    if (!searchParams.get("tab")) {
      router.replace(`/${params.org}/infra?tab=runners`);
    }
  }, [searchParams, router, params.org]);

  const handleBack = useCallback(() => {
    router.push(`/${params.org}/infra?tab=${tab}`);
  }, [router, params.org, tab]);

  return (
    <div className="h-full w-full overflow-auto">
      <div className="mx-auto max-w-[1400px] px-6 py-6">
        {tab === "runners" ? (
          <RunnerSection
            orgSlug={params.org}
            selectedId={selectedId}
            idMissing={!idParam}
            onBack={handleBack}
            t={t}
          />
        ) : (
          <RepoSection
            orgSlug={params.org}
            selectedId={selectedId}
            idMissing={!idParam}
            onBack={handleBack}
            t={t}
          />
        )}
      </div>
    </div>
  );
}

function RepoSection({
  orgSlug,
  selectedId,
  idMissing,
  onBack,
  t,
}: {
  orgSlug: string;
  selectedId: number;
  idMissing: boolean;
  onBack: () => void;
  t: (k: string) => string;
}) {
  const router = useRouter();
  const currentOrg = useCurrentOrg();
  const [loading, setLoading] = useState(true);
  const [firstId, setFirstId] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const raw = await getRepositoryService().list();
        if (cancelled) return;
        const parsed = JSON.parse(raw) as { repositories?: RepositoryData[] };
        setFirstId(parsed.repositories?.[0]?.id ?? null);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [currentOrg]);

  useEffect(() => {
    if (!idMissing || loading || firstId == null) return;
    router.replace(`/${orgSlug}/infra?tab=repositories&id=${firstId}`);
  }, [idMissing, loading, firstId, router, orgSlug]);

  if (loading) return <CenteredSpinner className="h-64" />;

  if (idMissing && firstId == null) {
    return (
      <EmptyState
        size="full"
        icon={<FolderGit2 className="h-12 w-12" />}
        title={t("repositories.emptyTitle")}
        description={t("repositories.emptyDescription")}
        actions={
          <Button onClick={() => router.push(`/${orgSlug}/infra?tab=repositories&import=1`)}>
            <Plus className="mr-1 h-4 w-4" />
            {t("repositories.import")}
          </Button>
        }
      />
    );
  }

  if (Number.isNaN(selectedId)) return null;
  return <InfraRepositoryDetail repositoryId={selectedId} onBack={onBack} />;
}

function RunnerSection({
  orgSlug,
  selectedId,
  idMissing,
  onBack,
  t,
}: {
  orgSlug: string;
  selectedId: number;
  idMissing: boolean;
  onBack: () => void;
  t: (k: string) => string;
}) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const currentOrg = useCurrentOrg();
  const runners = useRunners();
  const loading = useRunnerStore((s) => s.loading);
  const fetchRunners = useRunnerStore((s) => s.fetchRunners);
  const addRunnerOpen = searchParams.get("add") === "1";

  useEffect(() => {
    if (currentOrg) fetchRunners();
  }, [currentOrg, fetchRunners]);

  const firstId = runners[0]?.id ?? null;

  useEffect(() => {
    if (!idMissing || loading || firstId == null) return;
    router.replace(`/${orgSlug}/infra?tab=runners&id=${firstId}`);
  }, [idMissing, loading, firstId, router, orgSlug]);

  const closeAddRunnerModal = useCallback(() => {
    const next = new URLSearchParams(searchParams.toString());
    next.delete("add");
    if (!next.get("tab")) next.set("tab", "runners");
    router.replace(`/${orgSlug}/infra?${next.toString()}`);
  }, [orgSlug, router, searchParams]);

  const addRunnerModal = (
    <AddRunnerModal
      open={addRunnerOpen}
      onClose={closeAddRunnerModal}
      onCreated={closeAddRunnerModal}
    />
  );

  if (loading && runners.length === 0) {
    return (
      <>
        <CenteredSpinner className="h-64" />
        {addRunnerModal}
      </>
    );
  }

  if (idMissing && firstId == null) {
    return (
      <>
        <EmptyState
          size="full"
          icon={<Server className="h-12 w-12" />}
          title={t("runners.emptyState.title")}
          description={t("runners.emptyState.description")}
          actions={
            <Button onClick={() => router.push(`/${orgSlug}/infra?tab=runners&add=1`)}>
              <Plus className="mr-1 h-4 w-4" />
              {t("runners.addRunner")}
            </Button>
          }
        />
        {addRunnerModal}
      </>
    );
  }

  if (Number.isNaN(selectedId)) return null;
  return (
    <>
      <InfraRunnerDetail runnerId={selectedId} onBack={onBack} />
      {addRunnerModal}
    </>
  );
}
