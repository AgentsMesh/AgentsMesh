"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Search, Loader2, MessageSquare } from "lucide-react";
import { useTranslations } from "next-intl";
import { channelApi, type ChannelMessage } from "@/lib/api/facade/channel";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { cn } from "@/lib/utils";

interface MessageSearchModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channelId: number | null;
  channelName?: string;
}

const DEBOUNCE_MS = 300;

export function MessageSearchModal({
  open,
  onOpenChange,
  channelId,
  channelName,
}: MessageSearchModalProps) {
  const t = useTranslations("channels.search");
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<ChannelMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      setQuery("");
      setResults([]);
      setError(null);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  useEffect(() => {
    if (!open || !channelId) return;
    const q = query.trim();
    if (q.length < 1) {
      setResults([]);
      setError(null);
      return;
    }
    const handle = setTimeout(async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await channelApi.searchMessages(channelId, q);
        setResults(res.messages ?? []);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    }, DEBOUNCE_MS);
    return () => clearTimeout(handle);
  }, [query, open, channelId]);

  const handlePick = useCallback(
    (message: ChannelMessage) => {
      onOpenChange(false);
      requestAnimationFrame(() => {
        const el = document.querySelector(`[data-message-id="${message.id}"]`);
        if (el) {
          el.scrollIntoView({ behavior: "smooth", block: "center" });
          el.classList.add("ring-2", "ring-primary");
          setTimeout(() => el.classList.remove("ring-2", "ring-primary"), 1500);
        }
      });
    },
    [onOpenChange],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl" title={channelName ? t("titleFor", { channel: channelName }) : t("title")}>
        <div className="flex flex-col">
          <div className="border-b border-border px-4 py-3">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                ref={inputRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("placeholder")}
                className="pl-8"
                data-testid="message-search-input"
              />
            </div>
          </div>
          <div className="max-h-[55vh] overflow-y-auto">
            {loading && (
              <div className="flex justify-center py-6">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            )}
            {error && !loading && (
              <p className="px-4 py-6 text-center text-sm text-destructive">{error}</p>
            )}
            {!loading && !error && query.trim() && results.length === 0 && (
              <div className="flex flex-col items-center justify-center px-4 py-8 text-muted-foreground">
                <MessageSquare className="mb-2 h-8 w-8 opacity-30" />
                <p className="text-sm">{t("empty")}</p>
              </div>
            )}
            {results.map((msg) => (
              <button
                key={msg.id}
                type="button"
                onClick={() => handlePick(msg)}
                className={cn(
                  "flex w-full flex-col items-start gap-1 border-b border-border px-4 py-2.5 text-left",
                  "transition-colors hover:bg-muted/50",
                )}
                data-testid="message-search-result"
              >
                <div className="flex items-baseline gap-2 text-[11px] text-muted-foreground">
                  <span className="font-medium text-foreground">
                    {msg.sender_user?.name
                      ?? msg.sender_user?.username
                      ?? (msg.sender_pod_info ? getPodDisplayName(msg.sender_pod_info) : msg.sender_pod)
                      ?? "Unknown"}
                  </span>
                  <span>{new Date(msg.created_at).toLocaleString()}</span>
                </div>
                <p className="line-clamp-2 text-[13px] text-foreground">{msg.body}</p>
              </button>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default MessageSearchModal;
