"use client";

import React, { useEffect, useRef, useCallback, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";
import { cn } from "@/lib/utils";
import { terminalPool, useWorkspaceStore } from "@/stores/workspace";
import { podApi } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import {
  X,
  Maximize2,
  Minimize2,
  ExternalLink,
  RotateCcw,
  Square,
  Circle,
  Loader2,
  AlertCircle,
} from "lucide-react";

interface TerminalPaneProps {
  paneId: string;
  podKey: string;
  title: string;
  isActive: boolean;
  onClose?: () => void;
  onMaximize?: () => void;
  onPopout?: () => void;
  showHeader?: boolean;
  className?: string;
}

export function TerminalPane({
  paneId,
  podKey,
  title,
  isActive,
  onClose,
  onMaximize,
  onPopout,
  showHeader = true,
  className,
}: TerminalPaneProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const connectionRef = useRef<{ send: (data: string) => void; disconnect: () => void } | null>(null);
  const [isMaximized, setIsMaximized] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<"connecting" | "connected" | "disconnected" | "error">("connecting");
  const { terminalFontSize, setActivePane } = useWorkspaceStore();

  // Pod readiness state
  const [podStatus, setPodStatus] = useState<string>("unknown");
  const [isPodReady, setIsPodReady] = useState(false);
  const [podError, setPodError] = useState<string | null>(null);

  // Check Pod readiness before connecting
  useEffect(() => {
    let mounted = true;
    let pollInterval: ReturnType<typeof setInterval> | null = null;

    const checkPodStatus = async () => {
      try {
        const { pod } = await podApi.get(podKey);
        if (!mounted) return;

        setPodStatus(pod.status);

        if (pod.status === "running") {
          setIsPodReady(true);
          setPodError(null);
          if (pollInterval) {
            clearInterval(pollInterval);
            pollInterval = null;
          }
        } else if (pod.status === "failed" || pod.status === "terminated") {
          setIsPodReady(false);
          setPodError(`Pod ${pod.status}`);
          if (pollInterval) {
            clearInterval(pollInterval);
            pollInterval = null;
          }
        }
        // For "initializing" or "paused", continue polling
      } catch (error) {
        console.error("Failed to check pod status:", error);
        // Continue polling on error
      }
    };

    // Initial check
    checkPodStatus();

    // Poll every 1 second until Pod is ready or failed
    pollInterval = setInterval(checkPodStatus, 1000);

    return () => {
      mounted = false;
      if (pollInterval) {
        clearInterval(pollInterval);
      }
    };
  }, [podKey]);

  // Initialize terminal (only when Pod is ready)
  useEffect(() => {
    if (!terminalRef.current || xtermRef.current || !isPodReady) return;

    const term = new XTerm({
      cursorBlink: true,
      cursorStyle: "block",
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      fontSize: terminalFontSize,
      lineHeight: 1.2,
      theme: {
        background: "#1e1e1e",
        foreground: "#d4d4d4",
        cursor: "#d4d4d4",
        cursorAccent: "#1e1e1e",
        selectionBackground: "#264f78",
        black: "#000000",
        red: "#cd3131",
        green: "#0dbc79",
        yellow: "#e5e510",
        blue: "#2472c8",
        magenta: "#bc3fbc",
        cyan: "#11a8cd",
        white: "#e5e5e5",
        brightBlack: "#666666",
        brightRed: "#f14c4c",
        brightGreen: "#23d18b",
        brightYellow: "#f5f543",
        brightBlue: "#3b8eea",
        brightMagenta: "#d670d6",
        brightCyan: "#29b8db",
        brightWhite: "#e5e5e5",
      },
      allowProposedApi: true,
    });

    // Add addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    const searchAddon = new SearchAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.loadAddon(searchAddon);

    // Open terminal
    term.open(terminalRef.current);

    // Fit after a short delay to ensure container is sized
    setTimeout(() => {
      fitAddon.fit();
    }, 50);

    // Connect to WebSocket pool
    const handleMessage = (data: Uint8Array | string) => {
      if (data instanceof Uint8Array) {
        term.write(data);
      } else {
        term.write(data);
      }
    };

    connectionRef.current = terminalPool.connect(podKey, handleMessage);

    // Update connection status
    const checkStatus = () => {
      const status = terminalPool.getStatus(podKey);
      if (status !== "none") {
        setConnectionStatus(status);
      }
    };
    checkStatus();
    const statusInterval = setInterval(checkStatus, 1000);

    // Handle input
    term.onData((data) => {
      connectionRef.current?.send(data);
    });

    // Handle resize
    term.onResize(({ rows, cols }) => {
      terminalPool.sendResize(podKey, rows, cols);
    });

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Cleanup
    return () => {
      clearInterval(statusInterval);
      connectionRef.current?.disconnect();
      term.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
    };
  }, [podKey, terminalFontSize, isPodReady]);

  // Handle resize
  useEffect(() => {
    const handleResize = () => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
      }
    };

    // Observe container size changes
    const resizeObserver = new ResizeObserver(handleResize);
    if (terminalRef.current?.parentElement) {
      resizeObserver.observe(terminalRef.current.parentElement);
    }

    window.addEventListener("resize", handleResize);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  // Focus terminal when pane becomes active
  useEffect(() => {
    if (isActive && xtermRef.current) {
      xtermRef.current.focus();
      fitAddonRef.current?.fit();
    }
  }, [isActive]);

  // Update font size
  useEffect(() => {
    if (xtermRef.current) {
      xtermRef.current.options.fontSize = terminalFontSize;
      fitAddonRef.current?.fit();
    }
  }, [terminalFontSize]);

  const handleFocus = useCallback(() => {
    setActivePane(paneId);
  }, [paneId, setActivePane]);

  const handleMaximize = useCallback(() => {
    setIsMaximized(!isMaximized);
    onMaximize?.();
    // Fit terminal after layout change
    setTimeout(() => {
      fitAddonRef.current?.fit();
    }, 100);
  }, [isMaximized, onMaximize]);

  const getStatusColor = () => {
    switch (connectionStatus) {
      case "connected":
        return "text-green-500";
      case "connecting":
        return "text-yellow-500 animate-pulse";
      case "disconnected":
        return "text-gray-500";
      case "error":
        return "text-red-500";
      default:
        return "text-gray-500";
    }
  };

  return (
    <div
      className={cn(
        "flex flex-col h-full bg-[#1e1e1e] rounded-lg overflow-hidden border",
        isActive ? "border-primary" : "border-border",
        isMaximized && "fixed inset-4 z-50",
        className
      )}
      onClick={handleFocus}
    >
      {/* Header */}
      {showHeader && (
        <div className="h-8 flex items-center justify-between px-2 bg-[#252526] border-b border-[#3c3c3c]">
          <div className="flex items-center gap-2 min-w-0">
            <Circle className={cn("w-2 h-2 flex-shrink-0", getStatusColor())} />
            <span className="text-xs text-[#cccccc] truncate">{title}</span>
            <code className="text-[10px] text-[#808080] truncate">
              {podKey.substring(0, 8)}
            </code>
          </div>
          <div className="flex items-center gap-1 flex-shrink-0">
            {onPopout && (
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 hover:bg-[#3c3c3c] text-[#cccccc]"
                onClick={(e) => {
                  e.stopPropagation();
                  onPopout();
                }}
                title="Popout"
              >
                <ExternalLink className="w-3 h-3" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              className="h-5 w-5 p-0 hover:bg-[#3c3c3c] text-[#cccccc]"
              onClick={(e) => {
                e.stopPropagation();
                handleMaximize();
              }}
              title={isMaximized ? "Restore" : "Maximize"}
            >
              {isMaximized ? (
                <Minimize2 className="w-3 h-3" />
              ) : (
                <Maximize2 className="w-3 h-3" />
              )}
            </Button>
            {onClose && (
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 hover:bg-[#3c3c3c] text-[#cccccc] hover:text-red-400"
                onClick={(e) => {
                  e.stopPropagation();
                  onClose();
                }}
                title="Close"
              >
                <X className="w-3 h-3" />
              </Button>
            )}
          </div>
        </div>
      )}

      {/* Terminal or Loading/Error State */}
      {!isPodReady ? (
        <div className="flex-1 flex items-center justify-center bg-[#1e1e1e]">
          {podError ? (
            // Error state
            <div className="text-center p-4">
              <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-3" />
              <p className="text-[#cccccc] font-medium mb-1">{podError}</p>
              <p className="text-sm text-[#808080]">
                The pod cannot be connected. Please check the pod status or create a new one.
              </p>
            </div>
          ) : (
            // Waiting state
            <div className="text-center p-4">
              <Loader2 className="w-12 h-12 text-primary animate-spin mx-auto mb-3" />
              <p className="text-[#cccccc] font-medium mb-1">Waiting for Pod to be ready...</p>
              <p className="text-sm text-[#808080]">
                Status: <span className="text-yellow-500">{podStatus}</span>
              </p>
            </div>
          )}
        </div>
      ) : (
        <div
          ref={terminalRef}
          className="flex-1 min-h-0"
          style={{ minHeight: showHeader ? "calc(100% - 32px)" : "100%" }}
        />
      )}
    </div>
  );
}

export default TerminalPane;
