import { useState, useRef, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { ArrowLeft, Send, Paperclip, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  TicketStatusBadge,
  TicketCategoryBadge,
  TicketPriorityBadge,
} from "@/components/support/ticket-status-badge";
import { MessageList } from "@/components/support/message-list";
import {
  getSupportTicketDetail,
  addSupportTicketMessage,
  SupportTicketDetail,
} from "@/lib/api/support-ticket";

export function SupportTicketDetailPage() {
  const params = useParams();
  const router = useRouter();
  const t = useTranslations();
  const ticketId = Number(params.id);
  const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

  const [data, setData] = useState<SupportTicketDetail | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [replyContent, setReplyContent] = useState("");
  const [replyFiles, setReplyFiles] = useState<File[]>([]);
  const [isSending, setIsSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchDetail = useCallback(async () => {
    try {
      const result = await getSupportTicketDetail(ticketId);
      setData(result);
      setError(null);
    } catch {
      setError(t("support.error.loadFailed"));
    } finally {
      setIsLoading(false);
    }
  }, [ticketId, t]);

  useEffect(() => {
    fetchDetail();
    const interval = setInterval(() => {
      if (!document.hidden) {
        fetchDetail();
      }
    }, 15000);
    return () => clearInterval(interval);
  }, [fetchDetail]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [data?.messages?.length]);

  const handleSend = async () => {
    if (!replyContent.trim()) return;
    setIsSending(true);
    setSendError(null);
    try {
      await addSupportTicketMessage(
        ticketId,
        replyContent.trim(),
        replyFiles.length > 0 ? replyFiles : undefined
      );
      setReplyContent("");
      setReplyFiles([]);
      await fetchDetail();
    } catch {
      setSendError(t("support.error.sendFailed"));
    } finally {
      setIsSending(false);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const newFiles = Array.from(e.target.files);
      const oversized = newFiles.filter((f) => f.size > MAX_FILE_SIZE);
      if (oversized.length > 0) {
        toast.error(t("support.fileTooLarge", { max: "10MB" }));
        return;
      }
      setReplyFiles((prev) => [...prev, ...newFiles]);
    }
  };

  const removeFile = (index: number) => {
    setReplyFiles((prev) => prev.filter((_, i) => i !== index));
  };

  if (isLoading) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-6 space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-24 animate-pulse rounded-lg bg-muted/30" />
        <div className="h-64 animate-pulse rounded-lg bg-muted/30" />
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-6">
        <div className="flex flex-col items-center py-16">
          <p className="text-destructive">{error}</p>
          <Button variant="outline" onClick={fetchDetail} className="mt-4">
            {t("support.retry")}
          </Button>
        </div>
      </div>
    );
  }

  if (!data?.ticket) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-6">
        <div className="flex flex-col items-center py-16">
          <p className="text-muted-foreground">{t("support.notFound")}</p>
          <Button
            variant="outline"
            onClick={() => router.push("/support")}
            className="mt-4"
          >
            {t("support.backToList")}
          </Button>
        </div>
      </div>
    );
  }

  const { ticket, messages } = data;

  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <div className="flex items-start gap-3 mb-6">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.push("/support")}
          className="mt-0.5"
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1 min-w-0">
          <h1 className="text-lg font-bold">{ticket.title}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <TicketStatusBadge status={ticket.status} />
            <TicketCategoryBadge category={ticket.category} />
            <TicketPriorityBadge priority={ticket.priority} />
            <span className="text-xs text-muted-foreground">
              #{ticket.id} · {new Date(ticket.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-border bg-card">
        <div className="max-h-[500px] overflow-y-auto p-4">
          <MessageList messages={messages || []} />
          <div ref={messagesEndRef} />
        </div>

        {ticket.status !== "closed" && (
          <div className="border-t p-4">
            <div className="space-y-3">
              <textarea
                value={replyContent}
                onChange={(e) => setReplyContent(e.target.value)}
                placeholder={t("support.replyPlaceholder")}
                className="w-full min-h-[80px] rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-y"
                onKeyDown={(e) => {
                  if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                    handleSend();
                  }
                }}
              />

              {replyFiles.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {replyFiles.map((file, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs"
                    >
                      <Paperclip className="h-3 w-3" />
                      <span className="max-w-[150px] truncate">
                        {file.name}
                      </span>
                      <button
                        onClick={() => removeFile(i)}
                        className="ml-1 text-muted-foreground hover:text-foreground"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {sendError && (
                <p className="text-xs text-destructive">{sendError}</p>
              )}

              <div className="flex items-center justify-between">
                <div>
                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    accept="image/*,.pdf,.txt,.log"
                    onChange={handleFileSelect}
                    className="hidden"
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <Paperclip className="mr-1 h-4 w-4" />
                    {t("support.attach")}
                  </Button>
                </div>
                <Button
                  onClick={handleSend}
                  disabled={!replyContent.trim() || isSending}
                  size="sm"
                >
                  <Send className="mr-1 h-4 w-4" />
                  {isSending ? t("support.sending") : t("support.send")}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
