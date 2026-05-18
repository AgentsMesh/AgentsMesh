"use client";

import { useSearchParams, useRouter, useParams } from "next/navigation";
import { useEffect, useCallback } from "react";
import { RepoSection } from "./_components/RepoSection";
import { RunnerSection } from "./_components/RunnerSection";

type InfraTab = "repositories" | "runners";

export default function InfraPage() {
  const router = useRouter();
  const params = useParams<{ org: string }>();
  const searchParams = useSearchParams();

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
          />
        ) : (
          <RepoSection
            orgSlug={params.org}
            selectedId={selectedId}
            idMissing={!idParam}
            onBack={handleBack}
          />
        )}
      </div>
    </div>
  );
}

