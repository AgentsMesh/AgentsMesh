"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Copy, Check, MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useTranslations } from "next-intl";

interface MessageActionsProps {
  messageId: number;
  content: string;
  isOwnMessage: boolean;
  onEdit?: () => void;
  onDelete?: (messageId: number) => Promise<void>;
  onStartEdit: () => void;
}

export function MessageActions({
  messageId, content, isOwnMessage, onEdit, onDelete, onStartEdit,
}: MessageActionsProps) {
  const t = useTranslations("channels.messages");
  const [copied, setCopied] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch { /* Clipboard API not available */ }
  }, [content]);

  const handleDelete = useCallback(async () => {
    if (!onDelete) return;
    try { await onDelete(messageId); } catch { /* Error handled by store */ }
    setConfirmDelete(false);
  }, [onDelete, messageId]);

  return (
    <>
      <div className={`absolute -top-3 right-2 items-center gap-0.5 bg-background border border-border rounded-md shadow-sm z-10 px-0.5 ${menuOpen ? "flex" : "hidden group-hover/msg:flex"}`}>
        <Button variant="ghost" size="icon" className="h-6 w-6" aria-label={t("copyMessage")} onClick={handleCopy}>
          {copied ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}
        </Button>
        <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-6 w-6" aria-label={t("moreActions")}>
              <MoreHorizontal className="w-3 h-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={handleCopy}>
              <Copy className="w-3.5 h-3.5 mr-2" />{t("copyMessage")}
            </DropdownMenuItem>
            {isOwnMessage && onEdit && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={onStartEdit}>
                  <Pencil className="w-3.5 h-3.5 mr-2" />{t("editMessage")}
                </DropdownMenuItem>
              </>
            )}
            {isOwnMessage && onDelete && (
              <DropdownMenuItem onClick={() => setConfirmDelete(true)} className="text-destructive focus:text-destructive">
                <Trash2 className="w-3.5 h-3.5 mr-2" />{t("deleteMessage")}
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

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
    </>
  );
}
