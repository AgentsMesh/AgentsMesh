"use client";

import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { TicketDetail } from "@/components/tickets";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { ChevronRight, ArrowLeft } from "lucide-react";

export default function TicketDetailPage() {
  const params = useParams();
  const router = useRouter();
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const slug = params.slug as string;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
      {/* Breadcrumb */}
      <div className="mb-4 flex items-center gap-1.5">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
          onClick={() => router.back()}
        >
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <nav className="flex items-center gap-1 text-sm">
          <Link
            href={`/${currentOrg?.slug}/tickets`}
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            {t("tickets.title")}
          </Link>
          <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/50" />
          <span className="text-foreground font-mono text-xs font-medium bg-muted px-1.5 py-0.5 rounded">
            {slug}
          </span>
        </nav>
      </div>

      <TicketDetail slug={slug} />
    </div>
  );
}
