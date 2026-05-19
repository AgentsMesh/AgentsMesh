"use client";

import { useState, useEffect, useMemo } from "react";
import { getDemoFrames } from "./demoFrames";

export function useKanbanAnimation() {
  const [frameIndex, setFrameIndex] = useState(0);
  const [displayedLines, setDisplayedLines] = useState<number>(0);

  const frames = useMemo(() => getDemoFrames(), []);
  const currentFrame = frames[frameIndex];

  useEffect(() => {
    const nextFrame = frames[frameIndex + 1];
    if (nextFrame) {
      const delay = nextFrame.time - currentFrame.time;
      const timer = setTimeout(() => {
        setFrameIndex((prev) => prev + 1);
        setDisplayedLines(0);
      }, delay);
      return () => clearTimeout(timer);
    } else {
      const timer = setTimeout(() => {
        setFrameIndex(0);
        setDisplayedLines(0);
      }, 4000);
      return () => clearTimeout(timer);
    }
  }, [frameIndex, frames, currentFrame.time]);

  useEffect(() => {
    const maxLines = Math.max(
      ...currentFrame.terminals.map((t) => t.lines.length),
      0
    );
    if (displayedLines < maxLines) {
      const timer = setTimeout(() => {
        setDisplayedLines((prev) => prev + 1);
      }, 200);
      return () => clearTimeout(timer);
    }
  }, [frameIndex, displayedLines, currentFrame.terminals]);

  return { currentFrame, frameIndex, displayedLines };
}

export function getTerminalLineStyle(type: string): string {
  switch (type) {
    case "command":
      return "text-blue-400";
    case "output":
      return "text-muted-foreground";
    case "success":
      return "text-green-400";
    case "info":
      return "text-yellow-400";
    default:
      return "text-muted-foreground";
  }
}
