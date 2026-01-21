"use client";

import React, { useState, useEffect, useCallback, useRef } from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore, terminalPool } from "@/stores/workspace";
import {
  ChevronUp,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Keyboard,
  X,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n/client";

interface TerminalToolbarProps {
  className?: string;
}

// Special key codes - following Termux/Blink Shell conventions
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
  const { panes, activePane } = useWorkspaceStore();
  const [isOpen, setIsOpen] = useState(false);
  const [ctrlActive, setCtrlActive] = useState(false);
  const [altActive, setAltActive] = useState(false);
  const toolbarRef = useRef<HTMLDivElement>(null);

  const currentPane = panes.find((p) => p.id === activePane);

  // Close toolbar when clicking outside (touch-friendly)
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent | TouchEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('[data-terminal-toolbar]')) {
        setIsOpen(false);
      }
    };

    // Delay adding listener to avoid immediate close
    const timer = setTimeout(() => {
      document.addEventListener('click', handleClickOutside);
      document.addEventListener('touchend', handleClickOutside);
    }, 100);

    return () => {
      clearTimeout(timer);
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('touchend', handleClickOutside);
    };
  }, [isOpen]);

  // Send key with modifier support (Blink Shell style - modifiers stay active until key press)
  const sendKey = useCallback((key: string) => {
    if (!currentPane) return;

    let finalKey = key;

    // Apply Ctrl modifier for single character keys
    if (ctrlActive && key.length === 1) {
      const charCode = key.toUpperCase().charCodeAt(0) - 64;
      if (charCode >= 1 && charCode <= 26) {
        finalKey = String.fromCharCode(charCode);
      }
    }

    // Apply Alt modifier (ESC prefix for alt)
    if (altActive && key.length === 1) {
      finalKey = "\x1b" + key;
    }

    terminalPool.send(currentPane.podKey, finalKey);

    // Reset modifiers after sending (like Blink Shell)
    setCtrlActive(false);
    setAltActive(false);
  }, [currentPane, ctrlActive, altActive]);

  // Direct key send without modifiers
  const sendDirectKey = useCallback((key: string) => {
    if (!currentPane) return;
    terminalPool.send(currentPane.podKey, key);
    // Reset modifiers
    setCtrlActive(false);
    setAltActive(false);
  }, [currentPane]);

  if (!currentPane) {
    return null;
  }

  // Check if any modifier is active
  const hasActiveModifier = ctrlActive || altActive;

  return (
    <div className={cn("relative", className)} data-terminal-toolbar ref={toolbarRef}>
      {/* Floating trigger button - Blink Shell style */}
      {!isOpen && (
        <button
          onClick={() => setIsOpen(true)}
          className="absolute bottom-4 right-4 w-11 h-11 rounded-full bg-terminal-bg-secondary/90 backdrop-blur-sm text-terminal-text border border-terminal-border shadow-lg flex items-center justify-center hover:bg-terminal-bg-active active:scale-95 transition-all z-50"
          aria-label={t("terminalToolbar.open")}
        >
          <Keyboard className="w-5 h-5" />
        </button>
      )}

      {/* Compact two-row toolbar - Termux style */}
      {isOpen && (
        <div className="absolute bottom-0 left-0 right-0 bg-terminal-bg-secondary/95 backdrop-blur-sm border-t border-terminal-border shadow-lg animate-in slide-in-from-bottom-2 duration-150 safe-area-pb z-50">
          {/* Row 1: ESC, common shortcuts, navigation */}
          <div className="flex items-center gap-0.5 px-1 py-1 border-b border-terminal-border/50">
            <KeyButton label="ESC" onClick={() => sendDirectKey(KEYS.ESCAPE)} />
            <KeyButton
              label="^C"
              onClick={() => sendDirectKey(KEYS.CTRL_C)}
              variant="destructive"
              title="Ctrl+C"
            />
            <KeyButton label="^D" onClick={() => sendDirectKey(KEYS.CTRL_D)} title="Ctrl+D" />
            <KeyButton label="^Z" onClick={() => sendDirectKey(KEYS.CTRL_Z)} title="Ctrl+Z" />
            <div className="w-px h-5 bg-terminal-border/50 mx-0.5" />
            <KeyButton label="HOME" onClick={() => sendDirectKey(KEYS.HOME)} small />
            <KeyButton
              icon={<ChevronUp className="w-4 h-4" />}
              onClick={() => sendDirectKey(KEYS.UP)}
            />
            <KeyButton label="END" onClick={() => sendDirectKey(KEYS.END)} small />
            <KeyButton label="PGUP" onClick={() => sendDirectKey(KEYS.PAGE_UP)} small />
            {/* Spacer */}
            <div className="flex-1" />
            {/* Close button */}
            <button
              onClick={() => setIsOpen(false)}
              className="w-7 h-7 flex items-center justify-center text-terminal-text-muted hover:text-terminal-text hover:bg-terminal-bg-active rounded transition-colors"
              aria-label={t("terminalToolbar.close")}
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          {/* Row 2: TAB, modifiers, arrows, navigation */}
          <div className="flex items-center gap-0.5 px-1 py-1">
            <KeyButton label="TAB" onClick={() => sendKey(KEYS.TAB)} />
            <ModifierKey
              label="CTRL"
              active={ctrlActive}
              onClick={() => setCtrlActive(!ctrlActive)}
            />
            <ModifierKey
              label="ALT"
              active={altActive}
              onClick={() => setAltActive(!altActive)}
            />
            <div className="w-px h-5 bg-terminal-border/50 mx-0.5" />
            <KeyButton
              icon={<ChevronLeft className="w-4 h-4" />}
              onClick={() => sendDirectKey(KEYS.LEFT)}
            />
            <KeyButton
              icon={<ChevronDown className="w-4 h-4" />}
              onClick={() => sendDirectKey(KEYS.DOWN)}
            />
            <KeyButton
              icon={<ChevronRight className="w-4 h-4" />}
              onClick={() => sendDirectKey(KEYS.RIGHT)}
            />
            <KeyButton label="PGDN" onClick={() => sendDirectKey(KEYS.PAGE_DOWN)} small />
            {/* Spacer */}
            <div className="flex-1" />
            {/* Modifier indicator */}
            {hasActiveModifier && (
              <span className="text-[10px] text-primary font-medium px-1.5 py-0.5 bg-primary/10 rounded">
                {ctrlActive && "^"}{altActive && "⌥"}
              </span>
            )}
            {/* Common symbols for quick input */}
            <KeyButton label="|" onClick={() => sendKey("|")} />
            <KeyButton label="/" onClick={() => sendKey("/")} />
            <KeyButton label="-" onClick={() => sendKey("-")} />
            <KeyButton label="~" onClick={() => sendKey("~")} />
          </div>
        </div>
      )}
    </div>
  );
}

// Compact key button - Termux style
interface KeyButtonProps {
  label?: string;
  icon?: React.ReactNode;
  onClick: () => void;
  variant?: "default" | "destructive";
  title?: string;
  small?: boolean;
}

function KeyButton({
  label,
  icon,
  onClick,
  variant = "default",
  title,
  small,
}: KeyButtonProps) {
  return (
    <button
      className={cn(
        "h-8 min-w-[32px] px-1.5 rounded text-terminal-text font-mono",
        "flex items-center justify-center",
        "bg-terminal-bg-active/50 hover:bg-terminal-bg-active active:bg-terminal-bg-active/80",
        "border border-terminal-border/30 shadow-sm",
        "transition-colors duration-75",
        small && "text-[9px]",
        !small && "text-[11px]",
        variant === "destructive" && "text-red-400 hover:text-red-300 hover:bg-red-500/20"
      )}
      onClick={onClick}
      title={title}
    >
      {icon}
      {label && !icon && label}
    </button>
  );
}

// Modifier key with toggle state - Blink Shell style
interface ModifierKeyProps {
  label: string;
  active: boolean;
  onClick: () => void;
}

function ModifierKey({ label, active, onClick }: ModifierKeyProps) {
  return (
    <button
      className={cn(
        "h-8 min-w-[40px] px-2 rounded font-mono text-[11px]",
        "flex items-center justify-center",
        "border shadow-sm transition-all duration-75",
        active
          ? "bg-primary text-primary-foreground border-primary shadow-primary/30"
          : "bg-terminal-bg-active/50 text-terminal-text-muted border-terminal-border/30 hover:bg-terminal-bg-active hover:text-terminal-text"
      )}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

export default TerminalToolbar;
