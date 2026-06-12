import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";
import { InfraRunnerDetail } from "@/components/infra/InfraRunnerDetail";

// Thin shell over the web-tree InfraRunnerDetail (the same component the
// /:org/infra route renders). The previous parallel copy of the runner-detail
// hook/tabs drifted from the web tree (legacy facade calls, no resume error
// surface) — one implementation, two routes.
export function RunnerDetailPage() {
  const t = useTranslations();
  const { org, id } = useParams<{ org: string; id: string }>();
  const router = useRouter();
  const runnersTabHref = `/${org}/infra?tab=runners`;

  return (
    <div className="p-6 space-y-4">
      <Link href={runnersTabHref}>
        <Button variant="ghost" size="sm">
          <ArrowLeft className="w-4 h-4 mr-2" />
          {t("common.back")}
        </Button>
      </Link>
      <InfraRunnerDetail
        runnerId={Number(id)}
        onBack={() => router.push(runnersTabHref)}
      />
    </div>
  );
}
