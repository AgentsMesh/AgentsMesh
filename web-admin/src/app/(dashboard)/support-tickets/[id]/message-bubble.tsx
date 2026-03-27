import { Download, UserCircle, Shield } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { SupportTicketMessage } from "@/lib/api/admin";
import { getSupportTicketAttachmentUrl } from "@/lib/api/admin";
import { formatRelativeTime } from "@/lib/utils";

export function MessageBubble({ message }: { message: SupportTicketMessage }) {
  const isAdmin = message.is_admin_reply;

  const handleDownload = async (attachmentId: number, fileName: string) => {
    try {
      const { url } = await getSupportTicketAttachmentUrl(attachmentId);
      const a = document.createElement("a");
      a.href = url;
      a.download = fileName;
      a.target = "_blank";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    } catch {
      console.error("Failed to download attachment");
    }
  };

  return (
    <div className={`flex gap-3 ${isAdmin ? "flex-row-reverse" : ""}`}>
      <div className="flex-shrink-0 mt-1">
        {isAdmin ? (
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/20">
            <Shield className="h-4 w-4 text-primary" />
          </div>
        ) : (
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted">
            <UserCircle className="h-4 w-4 text-muted-foreground" />
          </div>
        )}
      </div>
      <div className={`flex-1 max-w-[80%] ${isAdmin ? "text-right" : ""}`}>
        <MessageHeader message={message} isAdmin={isAdmin} />
        <div className={`rounded-lg px-3 py-2 text-sm ${isAdmin ? "bg-primary/10 ml-auto" : "bg-muted"}`}>
          <p className="whitespace-pre-wrap text-left">{message.content}</p>
        </div>
        {message.attachments && message.attachments.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-2">
            {message.attachments.map((att) => (
              <button
                key={att.id}
                onClick={() => handleDownload(att.id, att.original_name)}
                className="flex items-center gap-1 rounded border px-2 py-1 text-xs hover:bg-muted transition-colors"
              >
                <Download className="h-3 w-3" />
                <span className="max-w-[120px] truncate">{att.original_name}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function MessageHeader({ message, isAdmin }: { message: SupportTicketMessage; isAdmin: boolean }) {
  return (
    <div className="flex items-center gap-2 mb-1">
      <span className="text-xs font-medium">
        {message.user?.name || message.user?.email || (isAdmin ? "Admin" : "User")}
      </span>
      {isAdmin && <Badge variant="outline" className="text-[10px] px-1 py-0">Admin</Badge>}
      <span className="text-xs text-muted-foreground">{formatRelativeTime(message.created_at)}</span>
    </div>
  );
}
