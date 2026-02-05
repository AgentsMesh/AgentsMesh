"use client";

import { useEffect, useState, lazy, Suspense, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, TicketStatus } from "@/stores/ticket";
import { StatusIcon, PriorityIcon, TypeIcon, getStatusDisplayInfo, getPriorityDisplayInfo, getTypeDisplayInfo } from "./TicketIcons";
import TicketPodPanel from "./TicketPodPanel";
import { useTicketExtraData } from "./hooks";
import { SubTicketsList, RelationsList, CommitsList, LabelsList } from "./shared";

// Lazy load BlockEditor to avoid SSR issues
const BlockEditor = lazy(() => import("@/components/ui/block-editor"));
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
        router.back();
      } catch (err) {
        console.error("Failed to delete ticket:", err);
      }
    }
  }, [confirm, deleteTicket, identifier, router, t]);

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
  const priorityInfo = getPriorityDisplayInfo(currentTicket.priority);
  const typeInfo = getTypeDisplayInfo(currentTicket.type);

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
            <div className="space-y-4">
              <input
                type="text"
                className="w-full text-2xl font-semibold px-3 py-2 border border-border rounded-md"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
              />
              <div>
                <label className="text-sm font-medium text-muted-foreground mb-1 block">
                  {t("tickets.detail.summary")}
                </label>
                <textarea
                  className="w-full px-3 py-2 border border-border rounded-md resize-none"
                  rows={2}
                  placeholder={t("tickets.createDialog.summaryPlaceholder")}
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                />
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground mb-1 block">
                  {t("tickets.detail.content")}
                </label>
                <div className="border border-border rounded-md overflow-hidden min-h-[200px] bg-card">
                  <Suspense fallback={<div className="h-[200px] animate-pulse bg-muted" />}>
                    <BlockEditor
                      initialContent={editContent}
                      onChange={setEditContent}
                      editable={true}
                    />
                  </Suspense>
                </div>
              </div>
              <div className="flex gap-2">
                <Button size="sm" onClick={handleSaveEdit}>{t("common.save")}</Button>
                <Button size="sm" variant="outline" onClick={() => setIsEditing(false)}>
                  {t("common.cancel")}
                </Button>
              </div>
            </div>
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
      <div className="lg:w-80 space-y-6">
        {/* Actions */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-3">{t("tickets.detail.actions")}</h3>
          <div className="space-y-2">
            <Button
              className="w-full"
              variant="outline"
              onClick={startEditing}
              disabled={isEditing}
            >
              {t("common.edit")}
            </Button>
            <Button
              className="w-full"
              variant="destructive"
              onClick={handleDelete}
            >
              {t("common.delete")}
            </Button>
          </div>
        </div>

        {/* Status */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-3">{t("tickets.filters.status")}</h3>
          <select
            className="w-full px-3 py-2 border border-border rounded-md bg-background text-sm"
            value={currentTicket.status}
            onChange={(e) => handleStatusChange(e.target.value as TicketStatus)}
          >
            <option value="backlog">{t("tickets.status.backlog")}</option>
            <option value="todo">{t("tickets.status.todo")}</option>
            <option value="in_progress">{t("tickets.status.in_progress")}</option>
            <option value="in_review">{t("tickets.status.in_review")}</option>
            <option value="done">{t("tickets.status.done")}</option>
            <option value="cancelled">{t("tickets.status.cancelled")}</option>
          </select>
        </div>

        {/* Details */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-3">{t("tickets.detail.details")}</h3>
          <dl className="space-y-3 text-sm">
            <div className="flex justify-between">
              <dt className="text-muted-foreground">{t("tickets.filters.type")}</dt>
              <dd className={`flex items-center gap-1 ${typeInfo.color}`}>
                <TypeIcon type={currentTicket.type} size="sm" />
                {t(`tickets.type.${currentTicket.type}`)}
              </dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-muted-foreground">{t("tickets.filters.priority")}</dt>
              <dd className={`flex items-center gap-1 ${priorityInfo.color}`}>
                <PriorityIcon priority={currentTicket.priority} size="sm" />
                {t(`tickets.priority.${currentTicket.priority}`)}
              </dd>
            </div>
            {currentTicket.due_date && (
              <div className="flex justify-between">
                <dt className="text-muted-foreground">{t("tickets.detail.dueDate")}</dt>
                <dd>{new Date(currentTicket.due_date).toLocaleDateString()}</dd>
              </div>
            )}
            {currentTicket.repository && (
              <div className="flex justify-between">
                <dt className="text-muted-foreground">{t("tickets.detail.repository")}</dt>
                <dd>{(currentTicket.repository as { name: string }).name}</dd>
              </div>
            )}
          </dl>
        </div>

        {/* Assignees */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-3">{t("tickets.detail.assignees")}</h3>
          {currentTicket.assignees && currentTicket.assignees.length > 0 ? (
            <div className="space-y-2">
              {currentTicket.assignees.map((assignee: { id: number; name?: string; username: string }) => (
                <div key={assignee.id} className="flex items-center gap-2">
                  <div className="w-6 h-6 rounded-full bg-muted flex items-center justify-center text-xs">
                    {(assignee.name || assignee.username)[0].toUpperCase()}
                  </div>
                  <span className="text-sm">{assignee.name || assignee.username}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{t("tickets.detail.noAssignees")}</p>
          )}
        </div>

        {/* Timestamps */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-3">{t("tickets.detail.timestamps")}</h3>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-muted-foreground">{t("tickets.detail.created")}</dt>
              <dd>{new Date(currentTicket.created_at).toLocaleString()}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-muted-foreground">{t("tickets.detail.updated")}</dt>
              <dd>{new Date(currentTicket.updated_at).toLocaleString()}</dd>
            </div>
          </dl>
        </div>
      </div>

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
