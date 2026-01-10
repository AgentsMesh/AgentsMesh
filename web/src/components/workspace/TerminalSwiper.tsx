"use client";

import React, { useRef, useEffect, useState } from "react";
import { useDrag } from "@use-gesture/react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { TerminalPane } from "./TerminalPane";
import { Terminal as TerminalIcon, Plus, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TerminalSwiperProps {
  onAddNew?: () => void;
  className?: string;
}

export function TerminalSwiper({ onAddNew, className }: TerminalSwiperProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const {
    panes,
    activePane,
    mobileActiveIndex,
    setMobileActiveIndex,
    removePane,
  } = useWorkspaceStore();

  const [translateX, setTranslateX] = useState(0);
  const [isDragging, setIsDragging] = useState(false);

  // Handle swipe gesture
  const bind = useDrag(
    ({ movement: [mx], direction: [dx], velocity: [vx], last, cancel }) => {
      if (panes.length <= 1) return;

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
      axis: "x",
      filterTaps: true,
      rubberband: true,
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

  // Ensure index is valid
  useEffect(() => {
    if (mobileActiveIndex >= panes.length && panes.length > 0) {
      setMobileActiveIndex(panes.length - 1);
    }
  }, [panes.length, mobileActiveIndex, setMobileActiveIndex]);

  const currentPane = panes[mobileActiveIndex];

  if (panes.length === 0) {
    return (
      <div className={cn("flex-1 flex items-center justify-center bg-[#1e1e1e]", className)}>
        <div className="text-center p-6">
          <TerminalIcon className="w-16 h-16 mx-auto mb-4 text-[#3c3c3c]" />
          <h3 className="text-lg font-medium text-[#cccccc] mb-2">No terminals</h3>
          <p className="text-sm text-[#808080] mb-4">
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
      <div className="h-10 flex items-center justify-between px-3 bg-[#252526] border-b border-[#3c3c3c]">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-[#808080]"
          onClick={goToPrev}
          disabled={mobileActiveIndex === 0}
        >
          <ChevronLeft className="w-4 h-4" />
        </Button>

        <div className="flex items-center gap-2">
          <span className="text-sm text-[#cccccc] font-medium">
            {currentPane?.title || "Terminal"}
          </span>
          <span className="text-xs text-[#808080]">
            {mobileActiveIndex + 1} / {panes.length}
          </span>
        </div>

        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-[#808080]"
          onClick={goToNext}
          disabled={mobileActiveIndex === panes.length - 1}
        >
          <ChevronRight className="w-4 h-4" />
        </Button>
      </div>

      {/* Dots indicator */}
      {panes.length > 1 && (
        <div className="flex items-center justify-center gap-1.5 py-2 bg-[#252526]">
          {panes.map((pane, index) => (
            <button
              key={pane.id}
              className={cn(
                "w-1.5 h-1.5 rounded-full transition-colors",
                index === mobileActiveIndex
                  ? "bg-primary"
                  : "bg-[#3c3c3c] hover:bg-[#555555]"
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
        className="flex-1 touch-pan-y overflow-hidden"
        style={{
          transform: isDragging ? `translateX(${translateX}px)` : "none",
          transition: isDragging ? "none" : "transform 0.2s ease-out",
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
