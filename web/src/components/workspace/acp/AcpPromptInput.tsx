"use client";

import { useState, useCallback } from "react";
import { Send } from "lucide-react";
import { relayPool } from "@/stores/relayConnection";

interface AcpPromptInputProps {
  podKey: string;
}

export function AcpPromptInput({ podKey }: AcpPromptInputProps) {
  const [prompt, setPrompt] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSend = useCallback(() => {
    if (!prompt.trim() || sending) return;
    if (!relayPool.isConnected(podKey)) {
      setError("Not connected");
      return;
    }
    setSending(true);
    setError(null);
    try {
      relayPool.sendAcpCommand(podKey, { type: "prompt", prompt });
      setPrompt("");
    } finally {
      setSending(false);
    }
  }, [prompt, podKey, sending]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="border-t p-3">
      {error && (
        <div className="text-xs text-red-500 mb-1">{error}</div>
      )}
      <div className="flex items-end gap-2">
        <textarea
          value={prompt}
          onChange={(e) => { setPrompt(e.target.value); setError(null); }}
          onKeyDown={handleKeyDown}
          placeholder="Send instruction..."
          disabled={sending}
          className="flex-1 resize-none rounded-lg border bg-background p-2 text-sm min-h-[40px] max-h-[120px]"
          rows={1}
        />
        <button
          onClick={handleSend}
          disabled={sending || !prompt.trim()}
          className="rounded-lg bg-primary p-2 text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          <Send className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
