"use client";

import React, { useRef, useEffect, useState } from "react";
import { useDrag } from "@use-gesture/react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodTitle } from "@/hooks/usePodTitle";
import { useTerminalInput } from "@/hooks/useTerminalInput";
import { TerminalPane } from "./TerminalPane";
import { Terminal as TerminalIcon, Plus, ChevronLeft, ChevronRight, Scaling } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TerminalSwiperProps {
  onAddNew?: () => void;
  className?: string;
}

export function TerminalSwiper({ onAddNew, className }: TerminalSwiperProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const panes = useWorkspaceStore((s) => s.panes);
  const mobileActiveIndex = useWorkspaceStore((s) => s.mobileActiveIndex);
  const setMobileActiveIndex = useWorkspaceStore((s) => s.setMobileActiveIndex);
  const removePane = useWorkspaceStore((s) => s.removePane);
  const { syncSize } = useTerminalInput();

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
      <SwiperHeader
        podKey={currentPane?.podKey}
        mobileActiveIndex={mobileActiveIndex}
        paneCount={panes.length}
        onPrev={goToPrev}
        onNext={goToNext}
        onSyncSize={syncSize}
      />

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

/** Swiper header bar with title and navigation. */
function SwiperHeader({
  podKey,
  mobileActiveIndex,
  paneCount,
  onPrev,
  onNext,
  onSyncSize,
}: {
  podKey?: string;
  mobileActiveIndex: number;
  paneCount: number;
  onPrev: () => void;
  onNext: () => void;
  onSyncSize: () => void;
}) {
  return (
    <div className="h-10 flex items-center justify-between px-3 bg-terminal-bg-secondary border-b border-terminal-border">
      <Button
        variant="ghost"
        size="sm"
        className="h-7 w-7 p-0 text-terminal-text-muted"
        onClick={onPrev}
        disabled={mobileActiveIndex === 0}
      >
        <ChevronLeft className="w-4 h-4" />
      </Button>

      <div className="flex items-center gap-2">
        <span className="text-sm text-terminal-text font-medium">
          {podKey ? <SwiperPaneTitle podKey={podKey} /> : "Terminal"}
        </span>
        <span className="text-xs text-terminal-text-muted">
          {mobileActiveIndex + 1} / {paneCount}
        </span>
      </div>

      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-terminal-text-muted"
          onClick={onSyncSize}
          title="Sync terminal size"
        >
          <Scaling className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-terminal-text-muted"
          onClick={onNext}
          disabled={mobileActiveIndex === paneCount - 1}
        >
          <ChevronRight className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}

/** Reads pod title via usePodTitle hook. */
function SwiperPaneTitle({ podKey }: { podKey: string }) {
  const title = usePodTitle(podKey, "Terminal");
  return <>{title}</>;
}

export default TerminalSwiper;
