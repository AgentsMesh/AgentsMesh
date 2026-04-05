"use client";

import { useState, useCallback } from "react";
import { Send, StopCircle } from "lucide-react";
import { relayPool } from "@/stores/relayConnection";
import { useAcpSessionStore } from "@/stores/acpSession";

interface AcpPromptInputProps {
  podKey: string;
}

export function AcpPromptInput({ podKey }: AcpPromptInputProps) {
  const [prompt, setPrompt] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const sessionState = useAcpSessionStore((s) => s.sessions[podKey]?.state);
  const isProcessing = sessionState === "processing" || sessionState === "waiting_permission";

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

  const handleCancel = useCallback(() => {
    if (!relayPool.isConnected(podKey)) {
      setError("Not connected");
      return;
    }
    setError(null);
    relayPool.sendAcpCommand(podKey, { type: "cancel" });
  }, [podKey]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="border-t px-3 py-2">
      {error && (
        <div className="text-xs text-red-500 mb-1">{error}</div>
      )}
      <div className="flex items-center gap-2">
        <textarea
          value={prompt}
          onChange={(e) => { setPrompt(e.target.value); setError(null); }}
          onKeyDown={handleKeyDown}
          placeholder="Send instruction..."
          disabled={sending}
          className="flex-1 resize-none rounded-md border bg-background px-3 py-1.5 text-sm min-h-[36px] max-h-[120px] leading-[20px]"
          rows={1}
        />
        {isProcessing ? (
          <button
            onClick={handleCancel}
            className="shrink-0 rounded-md bg-red-600 h-[36px] w-[36px] flex items-center justify-center text-white hover:bg-red-700"
            title="Cancel"
          >
            <StopCircle className="h-4 w-4" />
          </button>
        ) : (
          <button
            onClick={handleSend}
            disabled={sending || !prompt.trim()}
            className="shrink-0 rounded-md bg-primary h-[36px] w-[36px] flex items-center justify-center text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Send className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}
