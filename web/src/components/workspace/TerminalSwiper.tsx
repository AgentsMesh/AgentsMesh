"use client";

import React, { useRef, useEffect, useState } from "react";
import { useDrag } from "@use-gesture/react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { TerminalPane } from "./TerminalPane";
import { Terminal as TerminalIcon, Plus, ChevronLeft, ChevronRight, RefreshCw } from "lucide-react";
import { terminalPool } from "@/stores/workspace";
import { Button } from "@/components/ui/button";

interface TerminalSwiperProps {
  onAddNew?: () => void;
  className?: string;
}

export function TerminalSwiper({ onAddNew, className }: TerminalSwiperProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const {
    panes,
    mobileActiveIndex,
    setMobileActiveIndex,
    removePane,
  } = useWorkspaceStore();

  const [translateX, setTranslateX] = useState(0);
  const [isDragging, setIsDragging] = useState(false);

  // Handle swipe gesture for switching between terminals
  // Uses "lock" axis mode to detect swipe direction first, then lock to that axis
  // This allows vertical scrolling in terminal while still supporting horizontal swipe to switch
  const bind = useDrag(
    ({ movement: [mx, my], direction: [dx], velocity: [vx], last, cancel, event }) => {
      if (panes.length <= 1) return;

      // Exclude terminal input area from swipe gestures to prevent input conflicts
      const target = event?.target as HTMLElement;
      if (target?.closest('.xterm-helper-textarea') || target?.closest('.xterm-screen')) {
        cancel();
        setTranslateX(0);
        setIsDragging(false);
        return;
      }

      // If vertical movement is greater than horizontal, cancel the gesture
      // This allows touch events to pass through to terminal for scrolling
      if (!last && Math.abs(my) > Math.abs(mx) * 1.2) {
        cancel();
        setTranslateX(0);
        setIsDragging(false);
        return;
      }

      setIsDragging(!last);

      if (last) {
        // Determine if we should change slide
        const threshold = 50;
        const velocityThreshold = 0.5;

        let newIndex = mobileActiveIndex;

        if (mx < -threshold || (vx > velocityThreshold && dx < 0)) {
          newIndex = Math.min(mobileActiveIndex + 1, panes.length - 1);
        } else if (mx > threshold || (vx > velocityThreshold && dx > 0)) {
          newIndex = Math.max(mobileActiveIndex - 1, 0);
        }

        setMobileActiveIndex(newIndex);
        setTranslateX(0);
      } else {
        // Limit drag range
        const maxDrag = 100;
        setTranslateX(Math.max(-maxDrag, Math.min(maxDrag, mx)));
      }
    },
    {
      axis: "lock", // Lock to first detected axis direction
      filterTaps: true,
      rubberband: true,
      threshold: 10, // Require 10px movement before starting gesture
    }
  );

  // Navigate to previous/next
  const goToPrev = () => {
    if (mobileActiveIndex > 0) {
      setMobileActiveIndex(mobileActiveIndex - 1);
    }
  };

  const goToNext = () => {
    if (mobileActiveIndex < panes.length - 1) {
      setMobileActiveIndex(mobileActiveIndex + 1);
    }
  };

  // Ensure index is valid after hydration or panes change
  useEffect(() => {
    if (panes.length === 0) return;

    if (mobileActiveIndex >= panes.length) {
      setMobileActiveIndex(panes.length - 1);
    } else if (mobileActiveIndex < 0) {
      setMobileActiveIndex(0);
    }
  }, [panes.length, mobileActiveIndex, setMobileActiveIndex]);

  const currentPane = panes[mobileActiveIndex];

  // Sync terminal size handler - we need to get xterm instance from TerminalPane
  // Since TerminalPane doesn't expose xterm ref, we'll use terminalPool.forceResize
  // with a reasonable default size based on container
  const handleSyncSize = () => {
    if (currentPane && containerRef.current) {
      // Get container dimensions and estimate terminal size
      // This is a fallback - ideally we'd get the actual xterm size
      // For now, just trigger a resize with the current stored size or estimate
      const ptySize = terminalPool.getPtySize(currentPane.podKey);
      if (ptySize) {
        terminalPool.forceResize(currentPane.podKey, ptySize.rows, ptySize.cols);
      } else {
        // Estimate based on container - using typical terminal font metrics
        const container = containerRef.current;
        const cols = Math.floor(container.clientWidth / 9); // ~9px per char at 14px font
        const rows = Math.floor(container.clientHeight / 17); // ~17px per line at 14px font
        if (cols > 0 && rows > 0) {
          terminalPool.forceResize(currentPane.podKey, rows, cols);
        }
      }
    }
  };

  if (panes.length === 0) {
    return (
      <div className={cn("flex-1 flex items-center justify-center bg-terminal-bg", className)}>
        <div className="text-center p-6">
          <TerminalIcon className="w-16 h-16 mx-auto mb-4 text-terminal-border" />
          <h3 className="text-lg font-medium text-terminal-text mb-2">No terminals</h3>
          <p className="text-sm text-terminal-text-muted mb-4">
            Open a pod to start a terminal session
          </p>
          {onAddNew && (
            <Button onClick={onAddNew}>
              <Plus className="w-4 h-4 mr-2" />
              Open Terminal
            </Button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Swipe indicator / pagination */}
      <div className="h-10 flex items-center justify-between px-3 bg-terminal-bg-secondary border-b border-terminal-border">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-terminal-text-muted"
          onClick={goToPrev}
          disabled={mobileActiveIndex === 0}
        >
          <ChevronLeft className="w-4 h-4" />
        </Button>

        <div className="flex items-center gap-2">
          <span className="text-sm text-terminal-text font-medium">
            {currentPane?.title || "Terminal"}
          </span>
          <span className="text-xs text-terminal-text-muted">
            {mobileActiveIndex + 1} / {panes.length}
          </span>
        </div>

        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0 text-terminal-text-muted"
            onClick={handleSyncSize}
            title="Sync terminal size"
          >
            <RefreshCw className="w-4 h-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0 text-terminal-text-muted"
            onClick={goToNext}
            disabled={mobileActiveIndex === panes.length - 1}
          >
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Dots indicator */}
      {panes.length > 1 && (
        <div className="flex items-center justify-center gap-1.5 py-2 bg-terminal-bg-secondary">
          {panes.map((pane, index) => (
            <button
              key={pane.id}
              className={cn(
                "w-1.5 h-1.5 rounded-full transition-colors",
                index === mobileActiveIndex
                  ? "bg-primary"
                  : "bg-terminal-border hover:bg-terminal-bg-active"
              )}
              onClick={() => setMobileActiveIndex(index)}
            />
          ))}
        </div>
      )}

      {/* Terminal container with swipe */}
      <div
        ref={containerRef}
        {...bind()}
        className="flex-1 overflow-hidden"
        style={{
          transform: isDragging ? `translateX(${translateX}px)` : "none",
          transition: isDragging ? "none" : "transform 0.2s ease-out",
          touchAction: "pan-y", // Allow vertical scrolling, capture horizontal for swipe
        }}
      >
        {currentPane && (
          <TerminalPane
            paneId={currentPane.id}
            podKey={currentPane.podKey}
            title={currentPane.title}
            isActive={true}
            onClose={() => removePane(currentPane.id)}
            showHeader={false}
            className="h-full rounded-none border-0"
          />
        )}
      </div>
    </div>
  );
}

export default TerminalSwiper;
