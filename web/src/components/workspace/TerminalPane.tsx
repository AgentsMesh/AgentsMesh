"use client";

import React, { useCallback, useState } from "react";
import "@xterm/xterm/css/xterm.css";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStatus, useTerminal, useTouchScroll } from "@/hooks";
import { Button } from "@/components/ui/button";
import {
  X,
  Maximize2,
  Minimize2,
  ExternalLink,
  Circle,
  Loader2,
  AlertCircle,
  RefreshCw,
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
  const [isMaximized, setIsMaximized] = useState(false);
  const { terminalFontSize, setActivePane } = useWorkspaceStore();

  // Pod status tracking
  const { podStatus, isPodReady, podError } = usePodStatus(podKey);

  // Terminal initialization and management
  const {
    terminalRef,
    xtermRef,
    fitAddonRef,
    connectionStatus,
    syncSize,
  } = useTerminal(podKey, terminalFontSize, isPodReady, isActive);

  // Mobile touch scrolling support
  useTouchScroll(terminalRef, xtermRef, isPodReady);

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
  }, [isMaximized, onMaximize, fitAddonRef]);

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
            <Button
              variant="ghost"
              size="sm"
              className="h-5 w-5 p-0 hover:bg-[#3c3c3c] text-[#cccccc]"
              onClick={(e) => {
                e.stopPropagation();
                syncSize();
              }}
              title="Sync terminal size"
            >
              <RefreshCw className="w-3 h-3" />
            </Button>
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
          className="flex-1 min-h-0 overflow-auto"
          style={{
            minHeight: showHeader ? "calc(100% - 32px)" : "100%",
            touchAction: "pan-y pinch-zoom", // Enable touch scrolling and zoom
          }}
        />
      )}
    </div>
  );
}

export default TerminalPane;
