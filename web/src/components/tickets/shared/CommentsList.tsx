"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Reply, Pencil, Trash2 } from "lucide-react";
import type { TicketComment } from "@/lib/api/ticketTypes";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useCurrentUser, useAuthStore } from "@/stores/auth";
import { CommentInput } from "./CommentInput";

interface CommentsListProps {
  comments: TicketComment[];
  onAddComment: (
    content: string,
    parentId?: number,
    mentions?: Array<{ user_id: number; username: string }>
  ) => Promise<void>;
  onUpdateComment: (
    commentId: number,
    content: string,
    mentions?: Array<{ user_id: number; username: string }>
  ) => Promise<void>;
  onDeleteComment: (commentId: number) => Promise<void>;
  className?: string;
}

const URL_REGEX = /(https?:\/\/[^\s<]+[^\s<.,:;"')\]!?])/g;

function renderContent(content: string) {
  // Split by @mentions and URLs
  const tokens = content.split(/(@\w+|https?:\/\/[^\s<]+[^\s<.,:;"')\]!?])/g);
  return tokens.map((token, i) => {
    if (token.startsWith("@")) {
      return (
        <span
          key={i}
          className="text-primary font-medium bg-primary/10 rounded px-0.5"
        >
          {token}
        </span>
      );
    }
    if (URL_REGEX.test(token)) {
      URL_REGEX.lastIndex = 0;
      return (
        <a
          key={i}
          href={token}
          target="_blank"
          rel="noopener noreferrer"
          className="text-primary underline underline-offset-2 hover:text-primary/80 break-all"
        >
          {token}
        </a>
      );
    }
    return <span key={i}>{token}</span>;
  });
}

function formatRelativeDate(dateString: string) {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffDay > 7) {
    return date.toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      ...(date.getFullYear() !== now.getFullYear() ? { year: "numeric" } : {}),
    });
  }
  if (diffDay > 0) return `${diffDay}d ago`;
  if (diffHr > 0) return `${diffHr}h ago`;
  if (diffMin > 0) return `${diffMin}m ago`;
  return "just now";
}

function Avatar({ src, name, size = "md" }: { src?: string; name: string; size?: "sm" | "md" }) {
  const sizeClass = size === "sm" ? "w-6 h-6 text-[10px]" : "w-8 h-8 text-xs";
  const ringClass = size === "sm" ? "ring-1 ring-border/20" : "ring-1 ring-border/30";

  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={src} alt="" className={`${sizeClass} rounded-full shrink-0 ${ringClass}`} />
    );
  }
  return (
    <div className={`${sizeClass} rounded-full bg-primary/10 flex items-center justify-center font-semibold text-primary shrink-0 ${ringClass}`}>
      {(name || "?")[0].toUpperCase()}
    </div>
  );
}

export function CommentsList({
  comments,
  onAddComment,
  onUpdateComment,
  onDeleteComment,
  className,
}: CommentsListProps) {
  const t = useTranslations();
  const user = useCurrentUser();
  const { dialogProps, confirm } = useConfirmDialog();
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

  const handleDelete = useCallback(
    async (commentId: number) => {
      const confirmed = await confirm({
        title: t("tickets.detail.delete"),
        description: t("tickets.detail.deleteCommentConfirm"),
        variant: "destructive",
        confirmText: t("common.delete"),
        cancelText: t("common.cancel"),
      });
      if (confirmed) {
        await onDeleteComment(commentId);
      }
    },
    [confirm, onDeleteComment, t]
  );

  const renderComment = (comment: TicketComment, isReply = false) => {
    const isAuthor = user?.id === comment.user_id;
    const isEdited =
      comment.updated_at !== comment.created_at &&
      new Date(comment.updated_at ?? '').getTime() -
        new Date(comment.created_at ?? '').getTime() >
        1000;
    const hasReplies = comment.replies && comment.replies.length > 0;

    return (
      <div key={comment.id} className={isReply ? "relative pl-5 ml-4" : ""}>
        {isReply && (
          <div className="absolute left-0 top-0 bottom-0 w-px bg-border/40" />
        )}
        <div className="rounded-xl border border-border/50 bg-card overflow-hidden">
          {/* Comment header */}
          <div className="flex items-center gap-2.5 px-4 py-2.5 bg-muted/30 border-b border-border/30">
            <Avatar
              src={comment.user?.avatar_url}
              name={comment.user?.name || comment.user?.username || "?"}
              size="sm"
            />
            <span className="text-sm font-semibold text-foreground">
              {comment.user?.name || comment.user?.username || "Unknown"}
            </span>
            <span className="text-xs text-muted-foreground/50">
              &middot;
            </span>
            <span
              className="text-xs text-muted-foreground/50"
              title={new Date(comment.created_at ?? '').toLocaleString()}
            >
              {formatRelativeDate(comment.created_at ?? '')}
            </span>
            {isEdited && (
              <span className="text-[10px] text-muted-foreground/40 italic">
                ({t("tickets.detail.edited")})
              </span>
            )}
          </div>

          {/* Comment body */}
          <div className="px-4 py-3">
            {editingId === comment.id ? (
              <CommentInput
                initialContent={comment.content}
                onSubmit={handleSaveEdit}
                onCancel={() => setEditingId(null)}
              />
            ) : (
              <div className="text-sm text-foreground/90 whitespace-pre-wrap leading-relaxed">
                {renderContent(comment.content)}
              </div>
            )}
          </div>

          {/* Comment actions */}
          {editingId !== comment.id && (
            <div className="flex items-center gap-1 px-4 pb-2.5">
              {!isReply && (
                <button
                  type="button"
                  onClick={() =>
                    setReplyTo({
                      id: comment.id,
                      username: comment.user?.username || "unknown",
                    })
                  }
                  className="flex items-center gap-1 text-xs text-muted-foreground/50 hover:text-foreground transition-colors px-2 py-1 rounded-md hover:bg-muted/50"
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
                    className="flex items-center gap-1 text-xs text-muted-foreground/50 hover:text-foreground transition-colors px-2 py-1 rounded-md hover:bg-muted/50"
                  >
                    <Pencil className="w-3 h-3" />
                    {t("tickets.detail.edit")}
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDelete(comment.id)}
                    className="flex items-center gap-1 text-xs text-muted-foreground/50 hover:text-destructive transition-colors px-2 py-1 rounded-md hover:bg-destructive/5"
                  >
                    <Trash2 className="w-3 h-3" />
                  </button>
                </>
              )}
            </div>
          )}
        </div>

        {/* Threaded replies */}
        {hasReplies && (
          <div className="mt-2 space-y-2">
            {comment.replies!.map((reply) => renderComment(reply, true))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className={className}>
      {/* Input at top — GitHub style */}
      <CommentInput
        onSubmit={handleAddComment}
        replyTo={replyTo || undefined}
        onCancelReply={() => setReplyTo(null)}
      />

      {/* Comment list */}
      {comments.length > 0 && (
        <div className="mt-4 space-y-3">
          {comments.map((comment) => renderComment(comment))}
        </div>
      )}

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

export default CommentsList;
