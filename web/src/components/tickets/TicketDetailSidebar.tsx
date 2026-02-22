"use client";

import { Ticket, TicketStatus } from "@/stores/ticket";
import { Button } from "@/components/ui/button";
import { PriorityIcon, TypeIcon, getPriorityDisplayInfo, getTypeDisplayInfo } from "./TicketIcons";
import { RepositorySelect } from "@/components/common/RepositorySelect";

interface TicketDetailSidebarProps {
  ticket: Ticket;
  isEditing: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onStatusChange: (status: TicketStatus) => void;
  onRepositoryChange: (repositoryId: number | null) => void;
  t: (key: string, params?: Record<string, string | number>) => string;
}

/**
 * Sidebar component for TicketDetail page
 * Contains: Actions, Status, Details, Assignees, Timestamps
 */
export function TicketDetailSidebar({
  ticket,
  isEditing,
  onEdit,
  onDelete,
  onStatusChange,
  onRepositoryChange,
  t,
}: TicketDetailSidebarProps) {
  const priorityInfo = getPriorityDisplayInfo(ticket.priority);
  const typeInfo = getTypeDisplayInfo(ticket.type);

  return (
    <div className="lg:w-80 space-y-6">
      {/* Actions */}
      <div className="border border-border rounded-lg p-4">
        <h3 className="font-medium mb-3">{t("tickets.detail.actions")}</h3>
        <div className="space-y-2">
          <Button
            className="w-full"
            variant="outline"
            onClick={onEdit}
            disabled={isEditing}
          >
            {t("common.edit")}
          </Button>
          <Button
            className="w-full"
            variant="destructive"
            onClick={onDelete}
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
          value={ticket.status}
          onChange={(e) => onStatusChange(e.target.value as TicketStatus)}
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
              <TypeIcon type={ticket.type} size="sm" />
              {t(`tickets.type.${ticket.type}`)}
            </dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-muted-foreground">{t("tickets.filters.priority")}</dt>
            <dd className={`flex items-center gap-1 ${priorityInfo.color}`}>
              <PriorityIcon priority={ticket.priority} size="sm" />
              {t(`tickets.priority.${ticket.priority}`)}
            </dd>
          </div>
          {ticket.due_date && (
            <div className="flex justify-between">
              <dt className="text-muted-foreground">{t("tickets.detail.dueDate")}</dt>
              <dd>{new Date(ticket.due_date).toLocaleDateString()}</dd>
            </div>
          )}
          <div>
            <dt className="text-muted-foreground mb-1">{t("tickets.detail.repository")}</dt>
            <dd>
              <RepositorySelect
                value={ticket.repository_id ?? null}
                onChange={(value) => onRepositoryChange(value)}
                placeholder={t("tickets.detail.noRepository")}
                className="text-sm"
              />
            </dd>
          </div>
        </dl>
      </div>

      {/* Assignees */}
      <AssigneesCard assignees={ticket.assignees} t={t} />

      {/* Timestamps */}
      <TimestampsCard
        createdAt={ticket.created_at}
        updatedAt={ticket.updated_at}
        t={t}
      />
    </div>
  );
}

/**
 * Assignees card component
 */
function AssigneesCard({
  assignees,
  t,
}: {
  assignees?: Ticket["assignees"];
  t: (key: string) => string;
}) {
  return (
    <div className="border border-border rounded-lg p-4">
      <h3 className="font-medium mb-3">{t("tickets.detail.assignees")}</h3>
      {assignees && assignees.length > 0 ? (
        <div className="space-y-2">
          {assignees.map((assignee) => (
            <div key={assignee.user_id} className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-muted flex items-center justify-center text-xs">
                {(assignee.user?.name || assignee.user?.username || "?")[0].toUpperCase()}
              </div>
              <span className="text-sm">{assignee.user?.name || assignee.user?.username}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">{t("tickets.detail.noAssignees")}</p>
      )}
    </div>
  );
}

/**
 * Timestamps card component
 */
function TimestampsCard({
  createdAt,
  updatedAt,
  t,
}: {
  createdAt: string;
  updatedAt: string;
  t: (key: string) => string;
}) {
  return (
    <div className="border border-border rounded-lg p-4">
      <h3 className="font-medium mb-3">{t("tickets.detail.timestamps")}</h3>
      <dl className="space-y-2 text-sm">
        <div className="flex justify-between">
          <dt className="text-muted-foreground">{t("tickets.detail.created")}</dt>
          <dd>{new Date(createdAt).toLocaleString()}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-muted-foreground">{t("tickets.detail.updated")}</dt>
          <dd>{new Date(updatedAt).toLocaleString()}</dd>
        </div>
      </dl>
    </div>
  );
}

export default TicketDetailSidebar;
