"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore, terminalPool } from "@/stores/workspace";
import { Button } from "@/components/ui/button";
import {
  ChevronUp,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Keyboard,
  CornerDownLeft,
  Delete,
  Space,
  Type,
  Command,
  Option,
  ArrowBigUp,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n/client";

interface TerminalToolbarProps {
  className?: string;
}

// Special key codes
const KEYS = {
  TAB: "\t",
  ENTER: "\r",
  ESCAPE: "\x1b",
  CTRL_C: "\x03",
  CTRL_D: "\x04",
  CTRL_Z: "\x1a",
  CTRL_L: "\x0c",
  UP: "\x1b[A",
  DOWN: "\x1b[B",
  RIGHT: "\x1b[C",
  LEFT: "\x1b[D",
  HOME: "\x1b[H",
  END: "\x1b[F",
  PAGE_UP: "\x1b[5~",
  PAGE_DOWN: "\x1b[6~",
  DELETE: "\x1b[3~",
  BACKSPACE: "\x7f",
};

export function TerminalToolbar({ className }: TerminalToolbarProps) {
  const t = useTranslations();
  const { panes, activePane, terminalFontSize, setTerminalFontSize } = useWorkspaceStore();
  const [ctrlActive, setCtrlActive] = useState(false);
  const [altActive, setAltActive] = useState(false);
  const [shiftActive, setShiftActive] = useState(false);
  const [showExtended, setShowExtended] = useState(false);

  const currentPane = panes.find((p) => p.id === activePane);

  const sendKey = (key: string) => {
    if (!currentPane) return;

    let finalKey = key;

    // Apply modifiers
    if (ctrlActive && key.length === 1) {
      // Convert to control character
      const charCode = key.toUpperCase().charCodeAt(0) - 64;
      if (charCode >= 1 && charCode <= 26) {
        finalKey = String.fromCharCode(charCode);
      }
    }

    terminalPool.send(currentPane.podKey, finalKey);

    // Reset modifiers after sending (except if they're sticky)
    setCtrlActive(false);
    setAltActive(false);
    setShiftActive(false);
  };

  const adjustFontSize = (delta: number) => {
    setTerminalFontSize(terminalFontSize + delta);
  };

  if (!currentPane) {
    return null;
  }

  return (
    <div
      className={cn(
        "bg-terminal-bg-secondary border-t border-terminal-border safe-area-pb",
        className
      )}
    >
      {/* Extended toolbar (show/hide) */}
      {showExtended && (
        <div className="p-2 border-b border-terminal-border">
          <div className="flex flex-wrap gap-1.5">
            {/* Common control sequences */}
            <ToolbarButton
              label="Ctrl+C"
              onClick={() => sendKey(KEYS.CTRL_C)}
              variant="destructive"
            />
            <ToolbarButton
              label="Ctrl+D"
              onClick={() => sendKey(KEYS.CTRL_D)}
            />
            <ToolbarButton
              label="Ctrl+Z"
              onClick={() => sendKey(KEYS.CTRL_Z)}
            />
            <ToolbarButton
              label="Ctrl+L"
              onClick={() => sendKey(KEYS.CTRL_L)}
              title={t("terminalToolbar.clearScreen")}
            />
            <ToolbarButton
              label="Home"
              onClick={() => sendKey(KEYS.HOME)}
            />
            <ToolbarButton
              label="End"
              onClick={() => sendKey(KEYS.END)}
            />
            <ToolbarButton
              label="PgUp"
              onClick={() => sendKey(KEYS.PAGE_UP)}
            />
            <ToolbarButton
              label="PgDn"
              onClick={() => sendKey(KEYS.PAGE_DOWN)}
            />
          </div>

          {/* Font size control */}
          <div className="flex items-center gap-2 mt-2 pt-2 border-t border-terminal-border">
            <span className="text-xs text-terminal-text-muted">{t("terminalToolbar.font")}:</span>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-terminal-text"
              onClick={() => adjustFontSize(-1)}
            >
              A-
            </Button>
            <span className="text-xs text-terminal-text min-w-[2ch] text-center">
              {terminalFontSize}
            </span>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-terminal-text"
              onClick={() => adjustFontSize(1)}
            >
              A+
            </Button>
          </div>
        </div>
      )}

      {/* Main toolbar */}
      <div className="flex items-center p-1.5 gap-1">
        {/* Toggle extended */}
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-9 w-9 p-0 text-terminal-text-muted",
            showExtended && "bg-terminal-bg-active text-terminal-text"
          )}
          onClick={() => setShowExtended(!showExtended)}
        >
          <Keyboard className="w-4 h-4" />
        </Button>

        {/* Modifiers */}
        <ModifierButton
          label="Ctrl"
          icon={<Command className="w-3 h-3" />}
          active={ctrlActive}
          onClick={() => setCtrlActive(!ctrlActive)}
        />
        <ModifierButton
          label="Alt"
          icon={<Option className="w-3 h-3" />}
          active={altActive}
          onClick={() => setAltActive(!altActive)}
        />
        <ModifierButton
          label="Shift"
          icon={<ArrowBigUp className="w-3 h-3" />}
          active={shiftActive}
          onClick={() => setShiftActive(!shiftActive)}
        />

        <div className="w-px h-6 bg-terminal-border mx-1" />

        {/* Common keys */}
        <ToolbarButton
          icon={<Type className="w-3.5 h-3.5" />}
          label="Tab"
          onClick={() => sendKey(KEYS.TAB)}
        />
        <ToolbarButton
          label="Esc"
          onClick={() => sendKey(KEYS.ESCAPE)}
        />

        {/* Arrow keys */}
        <div className="flex items-center gap-0.5 ml-auto">
          <ToolbarButton
            icon={<ChevronUp className="w-4 h-4" />}
            onClick={() => sendKey(KEYS.UP)}
            square
          />
          <ToolbarButton
            icon={<ChevronDown className="w-4 h-4" />}
            onClick={() => sendKey(KEYS.DOWN)}
            square
          />
          <ToolbarButton
            icon={<ChevronLeft className="w-4 h-4" />}
            onClick={() => sendKey(KEYS.LEFT)}
            square
          />
          <ToolbarButton
            icon={<ChevronRight className="w-4 h-4" />}
            onClick={() => sendKey(KEYS.RIGHT)}
            square
          />
        </div>
      </div>
    </div>
  );
}

interface ToolbarButtonProps {
  label?: string;
  icon?: React.ReactNode;
  onClick: () => void;
  variant?: "default" | "destructive";
  title?: string;
  square?: boolean;
}

function ToolbarButton({
  label,
  icon,
  onClick,
  variant = "default",
  title,
  square,
}: ToolbarButtonProps) {
  return (
    <Button
      variant="ghost"
      size="sm"
      className={cn(
        "h-9 text-terminal-text hover:bg-terminal-bg-active",
        square ? "w-9 p-0" : "px-2.5",
        variant === "destructive" && "text-red-500 dark:text-red-400 hover:text-red-400 dark:hover:text-red-300"
      )}
      onClick={onClick}
      title={title}
    >
      {icon}
      {label && <span className="text-xs ml-1">{label}</span>}
    </Button>
  );
}

interface ModifierButtonProps {
  label: string;
  icon: React.ReactNode;
  active: boolean;
  onClick: () => void;
}

function ModifierButton({ label, icon, active, onClick }: ModifierButtonProps) {
  return (
    <Button
      variant="ghost"
      size="sm"
      className={cn(
        "h-9 px-2 text-xs",
        active
          ? "bg-primary text-primary-foreground hover:bg-primary/90"
          : "text-terminal-text-muted hover:text-terminal-text hover:bg-terminal-bg-active"
      )}
      onClick={onClick}
    >
      {icon}
      <span className="ml-1">{label}</span>
    </Button>
  );
}

export default TerminalToolbar;
