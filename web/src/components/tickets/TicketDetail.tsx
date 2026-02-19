"use client";

import { useEffect, useState, lazy, Suspense, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, TicketStatus } from "@/stores/ticket";
import { StatusIcon, TypeIcon, getStatusDisplayInfo } from "./TicketIcons";
import TicketPodPanel from "./TicketPodPanel";
import { useTicketExtraData } from "./hooks";
import { SubTicketsList, RelationsList, CommitsList, LabelsList } from "./shared";
import { TicketDetailSidebar } from "./TicketDetailSidebar";
import { TicketEditForm } from "./TicketEditForm";

// Lazy load BlockViewer to avoid SSR issues
const BlockViewer = lazy(() =>
  import("@/components/ui/block-editor").then((mod) => ({ default: mod.BlockViewer }))
);

interface TicketDetailProps {
  identifier: string;
}

export function TicketDetail({ identifier }: TicketDetailProps) {
  const router = useRouter();
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const { currentTicket, fetchTicket, updateTicket, updateTicketStatus, deleteTicket, loading, error } = useTicketStore();

  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editContent, setEditContent] = useState("");

  // Confirm dialog for delete
  const { dialogProps, confirm } = useConfirmDialog();

  // Use shared hook for extra data
  const { subTickets, relations, commits } = useTicketExtraData(identifier, !!currentTicket);

  // Fetch ticket data
  useEffect(() => {
    fetchTicket(identifier);
  }, [identifier, fetchTicket]);

  // Start editing - initialize edit fields from current ticket data
  const startEditing = () => {
    if (currentTicket) {
      setEditTitle(currentTicket.title);
      setEditDescription(currentTicket.description || "");
      setEditContent(currentTicket.content || "");
    }
    setIsEditing(true);
  };

  // Handle status change
  const handleStatusChange = async (newStatus: TicketStatus) => {
    try {
      await updateTicketStatus(identifier, newStatus);
    } catch (err) {
      console.error("Failed to update status:", err);
    }
  };

  // Handle save edit
  const handleSaveEdit = async () => {
    try {
      await updateTicket(identifier, {
        title: editTitle,
        description: editDescription,
        content: editContent,
      });
      setIsEditing(false);
    } catch (err) {
      console.error("Failed to update ticket:", err);
    }
  };

  // Handle delete with confirmation
  const handleDelete = useCallback(async () => {
    const confirmed = await confirm({
      title: t("tickets.detail.deleteTicket"),
      description: t("tickets.detail.deleteConfirmation", { identifier }),
      variant: "destructive",
      confirmText: t("common.delete"),
      cancelText: t("common.cancel"),
    });
    if (confirmed) {
      try {
        await deleteTicket(identifier);
        // Navigate to clean tickets list instead of router.back(),
        // which may return to a page that tries to reload the deleted ticket
        router.push(`/${currentOrg?.slug}/tickets`);
      } catch (err) {
        console.error("Failed to delete ticket:", err);
      }
    }
  }, [confirm, deleteTicket, identifier, router, currentOrg, t]);

  // Handle ticket click for sub-tickets and relations
  const handleTicketClick = (ticketIdentifier: string) => {
    router.push(`/${currentOrg?.slug}/tickets/${ticketIdentifier}`);
  };

  if (loading && !currentTicket) {
    return <TicketDetailSkeleton />;
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-red-600 dark:text-red-400 mb-4">{error}</div>
        <Button onClick={() => fetchTicket(identifier)}>{t("tickets.detail.retry")}</Button>
      </div>
    );
  }

  if (!currentTicket) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        {t("tickets.detail.notFound")}
      </div>
    );
  }

  const statusInfo = getStatusDisplayInfo(currentTicket.status);

  return (
    <div className="flex flex-col lg:flex-row gap-6">
      {/* Main Content */}
      <div className="flex-1 min-w-0">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-2">
            <TypeIcon type={currentTicket.type} size="md" />
            <span className="text-muted-foreground font-mono text-sm">
              {currentTicket.identifier}
            </span>
            <span className={`flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${statusInfo.bgColor} ${statusInfo.color}`}>
              <StatusIcon status={currentTicket.status} size="xs" />
              {statusInfo.label}
            </span>
          </div>

          {isEditing ? (
            <TicketEditForm
              title={editTitle}
              description={editDescription}
              content={editContent}
              onTitleChange={setEditTitle}
              onDescriptionChange={setEditDescription}
              onContentChange={setEditContent}
              onSave={handleSaveEdit}
              onCancel={() => setIsEditing(false)}
              t={t}
            />
          ) : (
            <>
              <h1 className="text-2xl font-semibold mb-2">{currentTicket.title}</h1>
              {currentTicket.description && (
                <p className="text-muted-foreground mb-4">
                  {currentTicket.description}
                </p>
              )}
              {currentTicket.content && (
                <div className="border border-border rounded-md overflow-hidden bg-card">
                  <Suspense fallback={<div className="h-[100px] animate-pulse bg-muted" />}>
                    <BlockViewer content={currentTicket.content} />
                  </Suspense>
                </div>
              )}
            </>
          )}
        </div>

        {/* Labels (using shared component) */}
        <LabelsList labels={currentTicket.labels || []} />

        {/* Sub-tickets (using shared component) */}
        <SubTicketsList
          subTickets={subTickets}
          onTicketClick={handleTicketClick}
        />

        {/* Relations (using shared component) */}
        <RelationsList
          relations={relations}
          onTicketClick={handleTicketClick}
        />

        {/* Commits (using shared component) */}
        <CommitsList commits={commits} />

        {/* AgentPods */}
        <TicketPodPanel
          ticketIdentifier={identifier}
          ticketTitle={currentTicket.title}
          ticketDescription={currentTicket.description}
          ticketId={currentTicket.id}
          repositoryId={currentTicket.repository_id}
        />
      </div>

      {/* Sidebar */}
      <TicketDetailSidebar
        ticket={currentTicket}
        isEditing={isEditing}
        onEdit={startEditing}
        onDelete={handleDelete}
        onStatusChange={handleStatusChange}
        t={t}
      />

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

function TicketDetailSkeleton() {
  return (
    <div className="animate-pulse" data-testid="ticket-detail-skeleton">
      <div className="flex flex-col lg:flex-row gap-6">
        <div className="flex-1">
          <div className="h-6 bg-muted rounded w-48 mb-4" />
          <div className="h-10 bg-muted rounded w-3/4 mb-4" />
          <div className="h-24 bg-muted rounded mb-6" />
          <div className="h-40 bg-muted rounded" />
        </div>
        <div className="lg:w-80 space-y-6">
          <div className="h-32 bg-muted rounded" />
          <div className="h-24 bg-muted rounded" />
          <div className="h-40 bg-muted rounded" />
        </div>
      </div>
    </div>
  );
}

export default TicketDetail;
