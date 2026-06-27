"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useMentionCandidates } from "@/hooks/useMentionCandidates";
import { getMessageReadBy } from "@/lib/api/facade/channelConnect";
import { readCurrentOrg } from "@/stores/auth";

interface ReadReceiptsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channelId: number;
  messageId: number;
}

/** "Read by" list for an own message — fetched on open, names resolved from the
 *  channel's mention candidates (cached member list). */
export function ReadReceiptsDialog({ open, onOpenChange, channelId, messageId }: ReadReceiptsDialogProps) {
  const t = useTranslations("channels.messages");
  const [userIds, setUserIds] = useState<number[] | null>(null);
  const { candidates } = useMentionCandidates({ channelId, enabled: open });

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    // Refetch on each open; messageId is fixed per dialog so the prior result
    // stays valid until this resolves (no synchronous reset needed).
    getMessageReadBy(readCurrentOrg()?.slug ?? "", channelId, messageId)
      .then((ids) => { if (!cancelled) setUserIds(ids); })
      .catch(() => { if (!cancelled) setUserIds([]); });
    return () => { cancelled = true; };
  }, [open, channelId, messageId]);

  const nameById = new Map<number, string>();
  for (const c of candidates) {
    const match = /^user:(\d+)$/.exec(c.id);
    if (match) nameById.set(Number(match[1]), c.displayName);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xs">
        <DialogHeader>
          <DialogTitle>{t("readByTitle", { count: userIds?.length ?? 0 })}</DialogTitle>
        </DialogHeader>
        {userIds == null ? null : userIds.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("readByNone")}</p>
        ) : (
          <ul className="flex max-h-64 flex-col gap-1 overflow-y-auto">
            {userIds.map((id) => (
              <li key={id} className="text-sm text-foreground">{nameById.get(id) ?? `#${id}`}</li>
            ))}
          </ul>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default ReadReceiptsDialog;
