"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  getSupportTicketDetail,
  replySupportTicket,
  updateSupportTicketStatus,
  assignSupportTicket,
  SupportTicketDetail,
} from "@/lib/api/admin";
import { useAuthStore } from "@/stores/auth";
import { formatRelativeTime } from "@/lib/utils";
import { TicketMetaCard } from "./ticket-meta-card";
import { MessageBubble } from "./message-bubble";
import { TicketReplyForm } from "./ticket-reply-form";

export default function SupportTicketDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { user } = useAuthStore();
  const ticketId = Number(params.id);

  const [data, setData] = useState<SupportTicketDetail | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [replyContent, setReplyContent] = useState("");
  const [replyFiles, setReplyFiles] = useState<File[]>([]);
  const [isSendingReply, setIsSendingReply] = useState(false);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);
  const [isAssigning, setIsAssigning] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const fetchDetail = useCallback(async () => {
    try {
      const result = await getSupportTicketDetail(ticketId);
      setData(result);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, [ticketId]);

  useEffect(() => {
    if (!ticketId) return;
    fetchDetail();
    const interval = setInterval(() => {
      if (!document.hidden) fetchDetail();
    }, 15000);
    return () => clearInterval(interval);
  }, [ticketId, fetchDetail]);

  const ticket = data?.ticket;
  const messages = data?.messages || [];

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length]);

  const handleReply = async () => {
    if (!replyContent.trim()) return;
    setIsSendingReply(true);
    try {
      await replySupportTicket(ticketId, replyContent, replyFiles.length > 0 ? replyFiles : undefined);
      setReplyContent("");
      setReplyFiles([]);
      toast.success("Reply sent successfully");
      await fetchDetail();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to send reply");
    } finally {
      setIsSendingReply(false);
    }
  };

  const handleStatusChange = async (status: string) => {
    setIsUpdatingStatus(true);
    try {
      await updateSupportTicketStatus(ticketId, status);
      toast.success("Status updated successfully");
      await fetchDetail();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to update status");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleAssign = async () => {
    setIsAssigning(true);
    try {
      await assignSupportTicket(ticketId, user?.id || 0);
      toast.success("Ticket assigned successfully");
      await fetchDetail();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to assign ticket");
    } finally {
      setIsAssigning(false);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-32 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded bg-muted" />
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-muted-foreground">Ticket not found</p>
        <Button variant="outline" onClick={() => router.push("/support-tickets")} className="mt-4">
          Back to list
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.push("/support-tickets")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-xl font-bold">{ticket.title}</h1>
          <div className="mt-1 flex items-center gap-2 text-sm text-muted-foreground">
            <span>#{ticket.id}</span>
            <span>·</span>
            <span>{ticket.user?.email || `User #${ticket.user_id}`}</span>
            <span>·</span>
            <span>{formatRelativeTime(ticket.created_at)}</span>
          </div>
        </div>
      </div>

      <TicketMetaCard
        ticket={ticket}
        isUpdatingStatus={isUpdatingStatus}
        isAssigning={isAssigning}
        onStatusChange={handleStatusChange}
        onAssign={handleAssign}
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Conversation</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4 max-h-[500px] overflow-y-auto pr-2">
            {messages.length === 0 ? (
              <p className="py-8 text-center text-muted-foreground">No messages yet</p>
            ) : (
              messages.map((msg) => <MessageBubble key={msg.id} message={msg} />)
            )}
            <div ref={messagesEndRef} />
          </div>
          <TicketReplyForm
            replyContent={replyContent}
            replyFiles={replyFiles}
            isSendingReply={isSendingReply}
            onContentChange={setReplyContent}
            onFilesChange={setReplyFiles}
            onSubmit={handleReply}
          />
        </CardContent>
      </Card>
    </div>
  );
}
