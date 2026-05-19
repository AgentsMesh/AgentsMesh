import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Plus, MessageSquare, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  TicketStatusBadge,
  TicketCategoryBadge,
  TicketPriorityBadge,
} from "@/components/support/ticket-status-badge";
import { CreateTicketDialog } from "@/components/support/create-ticket-dialog";
import {
  listSupportTickets,
  SupportTicket,
  SupportTicketListResponse,
} from "@/lib/api/support-ticket";
import { formatTimeAgo } from "@/lib/utils/time";
import { useEffect } from "react";

export function SupportPage() {
  const router = useRouter();
  const t = useTranslations();
  const [showCreate, setShowCreate] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<SupportTicketListResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTickets = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await listSupportTickets({
        status: statusFilter || undefined,
        page,
        page_size: 20,
      });
      setData(result);
    } catch {
      setError(t("support.error.loadFailed"));
    } finally {
      setIsLoading(false);
    }
  }, [statusFilter, page, t]);

  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  const tickets = data?.data || [];
  const totalPages = data?.total_pages || 1;

  const statusOptions = [
    { value: "", label: t("support.filter.all") },
    { value: "open", label: t("support.status.open") },
    { value: "in_progress", label: t("support.status.in_progress") },
    { value: "resolved", label: t("support.status.resolved") },
    { value: "closed", label: t("support.status.closed") },
  ];

  return (
    <div className="mx-auto max-w-4xl px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold">{t("support.title")}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t("support.description")}
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)} size="sm">
          <Plus className="mr-1 h-4 w-4" />
          {t("support.create")}
        </Button>
      </div>

      <div className="flex gap-2 mb-4 flex-wrap">
        {statusOptions.map((opt) => (
          <button
            key={opt.value}
            onClick={() => {
              setStatusFilter(opt.value);
              setPage(1);
            }}
            className={`rounded-full px-3 py-1 text-sm transition-colors ${
              statusFilter === opt.value
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-muted-foreground hover:bg-muted/80"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>

      <div className="space-y-3">
        {isLoading ? (
          Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="h-20 animate-pulse rounded-lg border border-border bg-muted/30"
            />
          ))
        ) : error ? (
          <div className="flex flex-col items-center py-16 text-center">
            <p className="text-destructive">{error}</p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={fetchTickets}
            >
              {t("support.retry")}
            </Button>
          </div>
        ) : tickets.length === 0 ? (
          <div className="flex flex-col items-center py-16 text-center">
            <MessageSquare className="h-10 w-10 text-muted-foreground/50 mb-3" />
            <p className="text-muted-foreground">{t("support.empty")}</p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => setShowCreate(true)}
            >
              {t("support.createFirst")}
            </Button>
          </div>
        ) : (
          tickets.map((ticket) => (
            <TicketCard
              key={ticket.id}
              ticket={ticket}
              onClick={() => router.push(`/support/${ticket.id}`)}
            />
          ))
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 mt-6">
          <Button
            variant="outline"
            size="icon"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <span className="text-sm text-muted-foreground">
            {page} / {totalPages}
          </span>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      )}

      <CreateTicketDialog
        open={showCreate}
        onOpenChange={setShowCreate}
        onCreated={fetchTickets}
      />
    </div>
  );
}

function TicketCard({
  ticket,
  onClick,
}: {
  ticket: SupportTicket;
  onClick: () => void;
}) {
  const t = useTranslations();

  return (
    <button
      onClick={onClick}
      className="w-full rounded-lg border border-border bg-card p-4 text-left hover:border-primary/30 transition-colors"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <h3 className="font-medium truncate">{ticket.title}</h3>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <TicketStatusBadge status={ticket.status} />
            <TicketCategoryBadge category={ticket.category} />
            <TicketPriorityBadge priority={ticket.priority} />
          </div>
        </div>
        <span className="text-xs text-muted-foreground whitespace-nowrap">
          {formatTimeAgo(ticket.created_at, t)}
        </span>
      </div>
    </button>
  );
}
