"use client";

import { useEffect, useState, useCallback, lazy, Suspense } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, Ticket, TicketStatus } from "@/stores/ticket";
import { StatusIcon, PriorityIcon, TypeIcon, getStatusDisplayInfo, getPriorityDisplayInfo, getTypeDisplayInfo } from "./TicketIcons";
import { ticketApi, TicketRelation, TicketCommit } from "@/lib/api/client";
import TicketPodPanel from "./TicketPodPanel";

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

  const [subTickets, setSubTickets] = useState<Ticket[]>([]);
  const [relations, setRelations] = useState<TicketRelation[]>([]);
  const [commits, setCommits] = useState<TicketCommit[]>([]);
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editContent, setEditContent] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [loadingExtra, setLoadingExtra] = useState(true);

  // Fetch ticket data
  useEffect(() => {
    fetchTicket(identifier);
  }, [identifier, fetchTicket]);

  // Fetch extra data (sub-tickets, relations, commits)
  const fetchExtraData = useCallback(async () => {
    if (!currentTicket) return;

    setLoadingExtra(true);
    try {
      const [subTicketsRes, relationsRes, commitsRes] = await Promise.all([
        ticketApi.getSubTickets(identifier).catch(() => ({ tickets: [] })),
        ticketApi.listRelations(identifier).catch(() => ({ relations: [] })),
        ticketApi.listCommits(identifier).catch(() => ({ commits: [] })),
      ]);

      setSubTickets(subTicketsRes.tickets || []);
      setRelations(relationsRes.relations || []);
      setCommits(commitsRes.commits || []);
    } catch (err) {
      console.error("Failed to fetch extra data:", err);
    } finally {
      setLoadingExtra(false);
    }
  }, [currentTicket, identifier]);

  useEffect(() => {
    if (currentTicket) {
      fetchExtraData();
      setEditTitle(currentTicket.title);
      setEditDescription(currentTicket.description || "");
      setEditContent(currentTicket.content || "");
    }
  }, [currentTicket, fetchExtraData]);

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

  // Handle delete
  const handleDelete = async () => {
    try {
      await deleteTicket(identifier);
      router.back();
    } catch (err) {
      console.error("Failed to delete ticket:", err);
    }
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

        {/* Labels */}
        {currentTicket.labels && currentTicket.labels.length > 0 && (
          <div className="flex flex-wrap gap-2 mb-6">
            {currentTicket.labels.map((label: { id: number; name: string; color: string }) => (
              <span
                key={label.id}
                className="px-2 py-1 rounded text-sm"
                style={{
                  backgroundColor: `${label.color}20`,
                  color: label.color,
                }}
              >
                {label.name}
              </span>
            ))}
          </div>
        )}

        {/* Sub-tickets */}
        {subTickets.length > 0 && (
          <div className="mb-6">
            <h3 className="font-medium mb-3 flex items-center gap-2">
              <span className="text-muted-foreground">◦</span>
              {t("tickets.detail.subTickets")} ({subTickets.length})
            </h3>
            <div className="border border-border rounded-lg divide-y divide-border">
              {subTickets.map((subTicket) => {
                const subStatusInfo = getStatusDisplayInfo(subTicket.status);
                return (
                  <div
                    key={subTicket.id}
                    className="px-4 py-3 hover:bg-muted/50 cursor-pointer"
                    onClick={() => router.push(`/${currentOrg?.slug}/tickets/${subTicket.identifier}`)}
                  >
                    <div className="flex items-center gap-2">
                      <TypeIcon type={subTicket.type} size="sm" />
                      <span className="font-mono text-xs text-muted-foreground">
                        {subTicket.identifier}
                      </span>
                      <span className="flex-1 truncate">{subTicket.title}</span>
                      <span className={`flex items-center gap-1 px-2 py-0.5 rounded text-xs ${subStatusInfo.bgColor} ${subStatusInfo.color}`}>
                        <StatusIcon status={subTicket.status} size="xs" />
                        {subStatusInfo.label}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Relations */}
        {relations.length > 0 && (
          <div className="mb-6">
            <h3 className="font-medium mb-3 flex items-center gap-2">
              <svg className="w-4 h-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
              </svg>
              {t("tickets.detail.related")} ({relations.length})
            </h3>
            <div className="border border-border rounded-lg divide-y divide-border">
              {relations.map((relation) => {
                const targetTicket = relation.target_ticket;
                if (!targetTicket) return null;
                return (
                  <div
                    key={relation.id}
                    className="px-4 py-3 hover:bg-muted/50 cursor-pointer"
                    onClick={() => router.push(`/${currentOrg?.slug}/tickets/${targetTicket.identifier}`)}
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground capitalize">
                        {relation.relation_type}
                      </span>
                      <span className="font-mono text-xs text-muted-foreground">
                        {targetTicket.identifier}
                      </span>
                      <span className="flex-1 truncate">{targetTicket.title}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Commits */}
        {commits.length > 0 && (
          <div className="mb-6">
            <h3 className="font-medium mb-3 flex items-center gap-2">
              <svg className="w-4 h-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              {t("tickets.detail.commits")} ({commits.length})
            </h3>
            <div className="border border-border rounded-lg divide-y divide-border">
              {commits.map((commit) => (
                <div key={commit.id} className="px-4 py-3">
                  <div className="flex items-start gap-3">
                    <code className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                      {commit.commit_sha.substring(0, 7)}
                    </code>
                    <div className="flex-1 min-w-0">
                      <p className="truncate">{commit.commit_message}</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        {commit.author_name} • {commit.committed_at ? new Date(commit.committed_at).toLocaleDateString() : "N/A"}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* AgentPods */}
        <TicketPodPanel
          ticketIdentifier={identifier}
          ticketTitle={currentTicket.title}
          ticketDescription={currentTicket.description}
          ticketId={currentTicket.id}
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
              onClick={() => setIsEditing(true)}
              disabled={isEditing}
            >
              {t("common.edit")}
            </Button>
            <Button
              className="w-full"
              variant="destructive"
              onClick={() => setShowDeleteConfirm(true)}
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

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-2">{t("tickets.detail.deleteTicket")}</h3>
            <p className="text-muted-foreground mb-4">
              {t("tickets.detail.deleteConfirmation", { identifier: currentTicket.identifier })}
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setShowDeleteConfirm(false)}>
                {t("common.cancel")}
              </Button>
              <Button variant="destructive" onClick={handleDelete}>
                {t("common.delete")}
              </Button>
            </div>
          </div>
        </div>
      )}
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
