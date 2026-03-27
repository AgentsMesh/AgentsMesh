"use client";

import { lazy, Suspense } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { X, ExternalLink, Clock, Loader2, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import TicketPodPanel from "./TicketPodPanel";
import { StatusSelect } from "./StatusSelect";
import { InlineEditableText } from "./InlineEditableText";
import { useTicketExtraData } from "./hooks";
import { SubTicketsList, RelationsList, CommitsList, LabelsList } from "./shared";
import { RepositorySelect } from "@/components/common/RepositorySelect";
import { useTicketPaneData } from "./useTicketPaneData";

const BlockViewer = lazy(() =>
  import("@/components/ui/block-editor").then((mod) => ({ default: mod.BlockViewer }))
);

export interface TicketDetailPaneProps {
  slug: string;
  onClose: () => void;
  className?: string;
}

export function TicketDetailPane({ slug, onClose, className }: TicketDetailPaneProps) {
  const t = useTranslations();
  const router = useRouter();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const { ticket, loading, error, handleStatusChange, handleTitleChange, handleRepositoryChange } = useTicketPaneData(slug);
  const { subTickets, relations, commits } = useTicketExtraData(slug, !!ticket);

  const handleTicketClick = (ticketSlug: string) => router.push(`/${currentOrg?.slug}/tickets?ticket=${ticketSlug}`);
  const formatDate = (ds: string) => new Date(ds).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });

  if (loading && !ticket) return <div className={cn("flex items-center justify-center h-full", className)}><Loader2 className="w-8 h-8 animate-spin text-muted-foreground" /></div>;
  if (error && !ticket) return <div className={cn("flex flex-col items-center justify-center h-full text-destructive", className)}><AlertCircle className="h-8 w-8 mb-2" /><p className="text-sm">{error}</p></div>;
  if (!ticket) return <div className={cn("flex items-center justify-center h-full text-muted-foreground", className)}><p className="text-sm">{t("tickets.detail.notFound")}</p></div>;

  return (
    <div className={cn("flex flex-col h-full bg-background", className)}>
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/40 shrink-0 bg-muted/20">
        <code className="text-xs text-muted-foreground/80 font-mono tracking-wide bg-muted/60 px-2 py-0.5 rounded">{ticket.slug}</code>
        <div className="flex items-center gap-0.5">
          <Link href={`/${currentOrg?.slug}/tickets/${ticket.slug}`}
            className="p-1.5 rounded-md hover:bg-muted/80 text-muted-foreground/60 hover:text-foreground transition-colors"
            title={t("tickets.detail.viewFullDetails")}><ExternalLink className="h-3.5 w-3.5" /></Link>
          <button onClick={onClose} className="p-1.5 rounded-md hover:bg-muted/80 text-muted-foreground/60 hover:text-foreground transition-colors"><X className="h-3.5 w-3.5" /></button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-4">
          <InlineEditableText value={ticket.title} onSave={handleTitleChange} placeholder={t("tickets.createDialog.titlePlaceholder")} className="text-lg font-bold leading-tight tracking-tight" inputClassName="text-lg font-bold tracking-tight" />
          <div className="flex items-center gap-2 flex-wrap"><StatusSelect value={ticket.status} onChange={handleStatusChange} size="sm" /></div>
          {ticket.content && (
            <div className="space-y-1.5">
              <label className="text-[11px] font-medium text-muted-foreground/60 uppercase tracking-wider">{t("tickets.detail.content")}</label>
              <div className="rounded-lg overflow-hidden bg-muted/20 ring-1 ring-border/30">
                <Suspense fallback={<div className="h-[100px] animate-pulse bg-muted/30 rounded-lg" />}><BlockViewer key={slug} content={ticket.content} /></Suspense>
              </div>
            </div>
          )}
          <LabelsList labels={ticket.labels || []} compact />
          {ticket.assignees && ticket.assignees.length > 0 && (
            <div className="flex items-center gap-2.5">
              <span className="text-[11px] font-medium text-muted-foreground/60 uppercase tracking-wider">{t("tickets.detail.assignees")}</span>
              <div className="flex items-center -space-x-1.5">
                {ticket.assignees.map((a) => (
                  <div key={a.user_id} className="w-6 h-6 rounded-full bg-primary/15 flex items-center justify-center text-[10px] font-semibold text-primary border-2 border-background ring-1 ring-primary/10"
                    title={a.user?.name || a.user?.username}>{(a.user?.name || a.user?.username || "?")[0].toUpperCase()}</div>
                ))}
              </div>
            </div>
          )}
          <div className="space-y-1.5">
            <label className="text-[11px] font-medium text-muted-foreground/60 uppercase tracking-wider">{t("tickets.detail.repository")}</label>
            <RepositorySelect value={ticket.repository_id ?? null} onChange={handleRepositoryChange} placeholder={t("tickets.detail.noRepository")} className="text-sm" />
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground/60">
            <span className="flex items-center gap-1.5"><Clock className="h-3 w-3" />{formatDate(ticket.created_at)}</span>
            {ticket.due_date && <span className={cn("flex items-center gap-1 font-medium", new Date(ticket.due_date) < new Date() ? "text-destructive" : "text-muted-foreground/70")}>Due {formatDate(ticket.due_date)}</span>}
          </div>
          {(subTickets.length > 0 || relations.length > 0 || commits.length > 0) && <div className="border-t border-border/30 pt-1" />}
          <SubTicketsList subTickets={subTickets} onTicketClick={handleTicketClick} compact />
          <RelationsList relations={relations} onTicketClick={handleTicketClick} compact />
          <CommitsList commits={commits} viewAllLink={`/${currentOrg?.slug}/tickets/${ticket.slug}`} compact />
          <TicketPodPanel ticketSlug={slug} ticketTitle={ticket.title} ticketId={ticket.id} repositoryId={ticket.repository_id} />
        </div>
      </div>

      <div className="shrink-0 px-4 py-2 border-t border-border/30">
        <Link href={`/${currentOrg?.slug}/tickets/${ticket.slug}`}>
          <Button variant="ghost" size="sm" className="w-full text-muted-foreground/70 hover:text-foreground text-xs">
            <ExternalLink className="h-3 w-3 mr-1.5" />{t("tickets.detail.viewFullDetails")}
          </Button>
        </Link>
      </div>
    </div>
  );
}

export default TicketDetailPane;
