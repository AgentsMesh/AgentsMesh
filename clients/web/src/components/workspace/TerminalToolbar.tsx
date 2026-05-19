"use client";

import React, { useState, useEffect, useCallback, useRef } from "react";
import { cn } from "@/lib/utils";
import { useTerminalInput } from "@/hooks/useTerminalInput";
import {
  ChevronUp,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Keyboard,
  X,
  ChevronsDown,
  CornerDownLeft,
} from "lucide-react";
import { useTranslations } from "next-intl";

interface TerminalToolbarProps {
  className?: string;
}

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
  const { activePodKey, send, scrollToBottom } = useTerminalInput();
  const [isOpen, setIsOpen] = useState(false);
  const [ctrlActive, setCtrlActive] = useState(false);
  const [altActive, setAltActive] = useState(false);
  const toolbarRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent | TouchEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('[data-terminal-toolbar]')) {
        setIsOpen(false);
      }
    };

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

  const sendKey = useCallback((key: string) => {
    let finalKey = key;

    if (ctrlActive && key.length === 1) {
      const charCode = key.toUpperCase().charCodeAt(0) - 64;
      if (charCode >= 1 && charCode <= 26) {
        finalKey = String.fromCharCode(charCode);
      }
    }

    if (altActive && key.length === 1) {
      finalKey = "\x1b" + key;
    }

    send(finalKey);

    setCtrlActive(false);
    setAltActive(false);
  }, [send, ctrlActive, altActive]);

  const sendDirectKey = useCallback((key: string) => {
    send(key);
    setCtrlActive(false);
    setAltActive(false);
  }, [send]);

  if (!activePodKey) {
    return null;
  }

  const hasActiveModifier = ctrlActive || altActive;

  return (
    <div className={cn("relative", className)} data-terminal-toolbar ref={toolbarRef}>
      {!isOpen && (
        <button
          onClick={() => setIsOpen(true)}
          className="absolute bottom-4 right-4 w-11 h-11 rounded-full bg-terminal-bg-secondary/90 backdrop-blur-sm text-terminal-text border border-terminal-border shadow-lg flex items-center justify-center hover:bg-terminal-bg-active active:scale-95 transition-all z-50"
          aria-label={t("terminalToolbar.open")}
        >
          <Keyboard className="w-5 h-5" />
        </button>
      )}

      {isOpen && (
        <div className="absolute bottom-0 left-0 right-0 bg-terminal-bg-secondary/95 backdrop-blur-sm border-t border-terminal-border shadow-lg animate-in slide-in-from-bottom-2 duration-150 safe-area-pb z-50">
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
            <KeyButton
              icon={<ChevronsDown className="w-4 h-4" />}
              onClick={scrollToBottom}
              title={t("terminalToolbar.scrollToBottom")}
            />
            <div className="flex-1" />
            <button
              onClick={() => setIsOpen(false)}
              className="w-7 h-7 flex items-center justify-center text-terminal-text-muted hover:text-terminal-text hover:bg-terminal-bg-active rounded transition-colors"
              aria-label={t("terminalToolbar.close")}
            >
              <X className="w-4 h-4" />
            </button>
          </div>

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
            <div className="flex-1" />
            {hasActiveModifier && (
              <span className="text-[10px] text-primary font-medium px-1.5 py-0.5 bg-primary/10 rounded">
                {ctrlActive && "^"}{altActive && "⌥"}
              </span>
            )}
            <KeyButton label="|" onClick={() => sendKey("|")} />
            <KeyButton label="/" onClick={() => sendKey("/")} />
            <KeyButton label="-" onClick={() => sendKey("-")} />
            <KeyButton label="~" onClick={() => sendKey("~")} />
            <KeyButton
              icon={<CornerDownLeft className="w-4 h-4" />}
              onClick={() => sendDirectKey(KEYS.ENTER)}
              title="Enter"
            />
          </div>
        </div>
      )}
    </div>
  );
}

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
