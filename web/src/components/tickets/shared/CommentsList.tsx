"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "next-intl";
import { MessageSquare, Reply, Pencil, Trash2 } from "lucide-react";
import { TicketComment } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { CommentInput } from "./CommentInput";

interface CommentsListProps {
  comments: TicketComment[];
  /** Called to create a new comment */
  onAddComment: (
    content: string,
    parentId?: number,
    mentions?: Array<{ user_id: number; username: string }>
  ) => Promise<void>;
  /** Called to update a comment */
  onUpdateComment: (
    commentId: number,
    content: string,
    mentions?: Array<{ user_id: number; username: string }>
  ) => Promise<void>;
  /** Called to delete a comment */
  onDeleteComment: (commentId: number) => Promise<void>;
  className?: string;
}

/**
 * Comments section component for ticket detail.
 * Supports threaded replies (one level), editing, deleting, and @mentions.
 */
export function CommentsList({
  comments,
  onAddComment,
  onUpdateComment,
  onDeleteComment,
  className,
}: CommentsListProps) {
  const t = useTranslations();
  const { user } = useAuthStore();
  const [replyTo, setReplyTo] = useState<{
    id: number;
    username: string;
  } | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);

  const handleAddComment = useCallback(
    async (
      content: string,
      mentions: Array<{ user_id: number; username: string }>
    ) => {
      await onAddComment(content, replyTo?.id, mentions);
      setReplyTo(null);
    },
    [onAddComment, replyTo]
  );

  const handleSaveEdit = useCallback(
    async (
      content: string,
      mentions: Array<{ user_id: number; username: string }>
    ) => {
      if (editingId === null) return;
      await onUpdateComment(editingId, content, mentions);
      setEditingId(null);
    },
    [editingId, onUpdateComment]
  );

  const handleDelete = async (commentId: number) => {
    if (!window.confirm(t("tickets.detail.deleteCommentConfirm"))) return;
    await onDeleteComment(commentId);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  /** Render content with @mention highlighting */
  const renderContent = (content: string) => {
    const parts = content.split(/(@\w+)/g);
    return parts.map((part, i) => {
      if (part.startsWith("@")) {
        return (
          <span
            key={i}
            className="text-primary font-medium bg-primary/10 rounded px-0.5"
          >
            {part}
          </span>
        );
      }
      return <span key={i}>{part}</span>;
    });
  };

  const renderComment = (comment: TicketComment, isReply = false) => {
    const isAuthor = user?.id === comment.user_id;
    const isEdited =
      comment.updated_at !== comment.created_at &&
      new Date(comment.updated_at).getTime() -
        new Date(comment.created_at).getTime() >
        1000;

    return (
      <div
        key={comment.id}
        className={`group ${isReply ? "ml-8 border-l-2 border-border pl-4" : ""}`}
      >
        <div className="flex items-start gap-3 py-3">
          {/* Avatar */}
          {comment.user?.avatar_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={comment.user.avatar_url}
              alt=""
              className="w-7 h-7 rounded-full shrink-0"
            />
          ) : (
            <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center text-xs font-medium text-primary shrink-0">
              {(comment.user?.username || "?")[0].toUpperCase()}
            </div>
          )}

          <div className="flex-1 min-w-0">
            {/* Header */}
            <div className="flex items-center gap-2 mb-1">
              <span className="text-sm font-medium">
                {comment.user?.name || comment.user?.username || "Unknown"}
              </span>
              <span className="text-xs text-muted-foreground">
                {formatDate(comment.created_at)}
              </span>
              {isEdited && (
                <span className="text-xs text-muted-foreground italic">
                  ({t("tickets.detail.edited")})
                </span>
              )}
            </div>

            {/* Content or Edit Input */}
            {editingId === comment.id ? (
              <CommentInput
                initialContent={comment.content}
                onSubmit={handleSaveEdit}
                onCancel={() => setEditingId(null)}
              />
            ) : (
              <div className="text-sm whitespace-pre-wrap">
                {renderContent(comment.content)}
              </div>
            )}

            {/* Actions */}
            {editingId !== comment.id && (
              <div className="flex items-center gap-1 mt-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
                {!isReply && (
                  <button
                    type="button"
                    onClick={() =>
                      setReplyTo({
                        id: comment.id,
                        username: comment.user?.username || "unknown",
                      })
                    }
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors px-1.5 py-0.5 rounded hover:bg-muted/50"
                  >
                    <Reply className="w-3 h-3" />
                    {t("tickets.detail.reply")}
                  </button>
                )}
                {isAuthor && (
                  <>
                    <button
                      type="button"
                      onClick={() => setEditingId(comment.id)}
                      className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors px-1.5 py-0.5 rounded hover:bg-muted/50"
                    >
                      <Pencil className="w-3 h-3" />
                      {t("tickets.detail.edit")}
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(comment.id)}
                      className="flex items-center gap-1 text-xs text-red-500 hover:text-red-600 transition-colors px-1.5 py-0.5 rounded hover:bg-red-50 dark:hover:bg-red-950/30"
                    >
                      <Trash2 className="w-3 h-3" />
                      {t("tickets.detail.delete")}
                    </button>
                  </>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Replies */}
        {comment.replies?.map((reply) => renderComment(reply, true))}
      </div>
    );
  };

  return (
    <div className={`mb-6 ${className || ""}`}>
      <h3 className="font-medium mb-3 flex items-center gap-2">
        <MessageSquare className="w-4 h-4 text-muted-foreground" />
        {t("tickets.detail.comments")}
        {comments.length > 0 && (
          <span className="text-muted-foreground text-sm font-normal">
            ({comments.length})
          </span>
        )}
      </h3>

      {/* Comment list */}
      {comments.length > 0 ? (
        <div className="border border-border rounded-lg divide-y divide-border px-4 mb-3">
          {comments.map((comment) => renderComment(comment))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground mb-3">
          {t("tickets.detail.noComments")}
        </p>
      )}

      {/* Comment input */}
      <CommentInput
        onSubmit={handleAddComment}
        replyTo={replyTo || undefined}
        onCancelReply={() => setReplyTo(null)}
      />
    </div>
  );
}

export default CommentsList;
