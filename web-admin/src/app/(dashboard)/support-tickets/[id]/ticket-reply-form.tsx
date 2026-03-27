import { useRef } from "react";
import { Send, Paperclip, X } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TicketReplyFormProps {
  replyContent: string;
  replyFiles: File[];
  isSendingReply: boolean;
  onContentChange: (content: string) => void;
  onFilesChange: (files: File[]) => void;
  onSubmit: () => void;
}

export function TicketReplyForm({
  replyContent,
  replyFiles,
  isSendingReply,
  onContentChange,
  onFilesChange,
  onSubmit,
}: TicketReplyFormProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      onFilesChange([...replyFiles, ...Array.from(e.target.files)]);
    }
  };

  const removeFile = (index: number) => {
    onFilesChange(replyFiles.filter((_, i) => i !== index));
  };

  return (
    <div className="mt-6 border-t pt-4">
      <div className="space-y-3">
        <textarea
          value={replyContent}
          onChange={(e) => onContentChange(e.target.value)}
          placeholder="Type your reply..."
          className="w-full min-h-[100px] rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-y"
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
              onSubmit();
            }
          }}
        />

        {replyFiles.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {replyFiles.map((file, i) => (
              <div key={i} className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs">
                <Paperclip className="h-3 w-3" />
                <span className="max-w-[150px] truncate">{file.name}</span>
                <button onClick={() => removeFile(i)} className="ml-1 text-muted-foreground hover:text-foreground">
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
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
            <Button variant="ghost" size="sm" onClick={() => fileInputRef.current?.click()}>
              <Paperclip className="mr-1 h-4 w-4" />
              Attach
            </Button>
          </div>
          <Button
            onClick={onSubmit}
            disabled={!replyContent.trim() || isSendingReply}
            size="sm"
          >
            <Send className="mr-1 h-4 w-4" />
            {isSendingReply ? "Sending..." : "Send Reply"}
          </Button>
        </div>
      </div>
    </div>
  );
}
