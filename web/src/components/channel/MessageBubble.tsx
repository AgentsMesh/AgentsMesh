"use client";

import { useState, useCallback } from "react";
import { Markdown } from "@/components/ui/markdown";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Copy, Check, MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useTranslations } from "next-intl";
import type { TransformedMessage } from "./types";

interface MessageBubbleProps {
  message: TransformedMessage;
  /** Whether this is the first message in a sender group (shows full header) */
  isFirstInGroup: boolean;
  formatTime: (dateString: string) => string;
  /** Current user ID for showing edit/delete actions */
  currentUserId?: number;
  /** Callback to edit a message */
  onEdit?: (messageId: number, content: string) => Promise<void>;
  /** Callback to delete a message */
  onDelete?: (messageId: number) => Promise<void>;
}

/**
 * Single message content renderer with Discord-style hover action bar.
 * Shows copy + more actions on hover in the top-right corner.
 */
export function MessageBubble({
  message,
  isFirstInGroup,
  formatTime,
  currentUserId,
  onEdit,
  onDelete,
}: MessageBubbleProps) {
  const t = useTranslations("channels.messages");
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(message.content);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const isOwnMessage = currentUserId != null && message.user?.id === currentUserId;

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(message.content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API not available
    }
  }, [message.content]);

  const handleEditSubmit = useCallback(async () => {
    if (!onEdit || editContent.trim() === "" || editContent === message.content) {
      setEditing(false);
      return;
    }
    try {
      await onEdit(message.id, editContent.trim());
      setEditing(false);
    } catch {
      // Error handled by store
    }
  }, [onEdit, editContent, message.id, message.content]);

  const handleEditKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleEditSubmit();
      }
      if (e.key === "Escape") {
        setEditing(false);
        setEditContent(message.content);
      }
    },
    [handleEditSubmit, message.content]
  );

  const handleDelete = useCallback(async () => {
    if (!onDelete) return;
    try {
      await onDelete(message.id);
    } catch {
      // Error handled by store
    }
    setConfirmDelete(false);
  }, [onDelete, message.id]);

  const isCode = message.messageType === "code";
  const isCommand = message.messageType === "command";

  return (
    <div className="group/msg relative">
      {/* Hover action bar — top-right corner */}
      <div className={`absolute -top-3 right-2 items-center gap-0.5 bg-background border border-border rounded-md shadow-sm z-10 px-0.5 ${menuOpen ? "flex" : "hidden group-hover/msg:flex"}`}>
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6"
          onClick={handleCopy}
        >
          {copied ? (
            <Check className="w-3 h-3 text-green-500" />
          ) : (
            <Copy className="w-3 h-3" />
          )}
        </Button>
        <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <MoreHorizontal className="w-3 h-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={handleCopy}>
              <Copy className="w-3.5 h-3.5 mr-2" />
              {t("copyMessage")}
            </DropdownMenuItem>
            {isOwnMessage && onEdit && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => { setEditing(true); setEditContent(message.content); }}>
                  <Pencil className="w-3.5 h-3.5 mr-2" />
                  {t("editMessage")}
                </DropdownMenuItem>
              </>
            )}
            {isOwnMessage && onDelete && (
              <DropdownMenuItem
                onClick={() => setConfirmDelete(true)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="w-3.5 h-3.5 mr-2" />
                {t("deleteMessage")}
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Message content */}
      <div className="flex items-start gap-3">
        {/* Time gutter — for non-first messages, show time on hover */}
        {!isFirstInGroup && (
          <span className="w-8 flex-shrink-0 text-[10px] text-muted-foreground opacity-0 group-hover/msg:opacity-100 transition-opacity pt-1 text-center tabular-nums">
            {formatTime(message.createdAt)}
          </span>
        )}

        <div className={`flex-1 min-w-0 ${isFirstInGroup ? "" : ""}`}>
          {editing ? (
            <div className="space-y-2">
              <Textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                onKeyDown={handleEditKeyDown}
                className="min-h-[60px] text-sm"
                autoFocus
              />
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Button size="sm" variant="default" className="h-6 text-xs" onClick={handleEditSubmit}>
                  {t("save")}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 text-xs"
                  onClick={() => { setEditing(false); setEditContent(message.content); }}
                >
                  {t("cancel")}
                </Button>
                <span>{t("editHint")}</span>
              </div>
            </div>
          ) : isCode ? (
            <pre className="p-3 bg-muted rounded-md text-sm overflow-x-auto">
              <code>{message.content}</code>
            </pre>
          ) : isCommand ? (
            <div className="p-2 bg-muted rounded-md text-sm font-mono text-green-600 dark:text-green-400">
              $ {message.content}
            </div>
          ) : (
            <div>
              <Markdown
                content={message.content}
                compact
                highlightMentions
                className="text-sm [&_p:first-child]:mt-0 [&_p:last-child]:mb-0"
              />
              {message.editedAt && (
                <span className="text-[10px] text-muted-foreground ml-1">
                  ({t("edited")})
                </span>
              )}
            </div>
          )}

          {/* Metadata */}
          {message.metadata && Object.keys(message.metadata).length > 0 && (
            <div className="mt-1.5 text-xs text-muted-foreground">
              <details>
                <summary className="cursor-pointer hover:text-foreground">
                  Metadata
                </summary>
                <pre className="mt-1 p-2 bg-muted rounded text-xs overflow-x-auto">
                  {JSON.stringify(message.metadata, null, 2)}
                </pre>
              </details>
            </div>
          )}
        </div>
      </div>

      {/* Delete confirmation dialog */}
      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteConfirmTitle")}</AlertDialogTitle>
            <AlertDialogDescription>{t("deleteConfirmDescription")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              {t("delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export default MessageBubble;
