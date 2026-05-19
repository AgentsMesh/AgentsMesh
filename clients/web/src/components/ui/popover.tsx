"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";

interface PopoverContextValue {
  open: boolean;
  setOpen: (open: boolean) => void;
  triggerRef: React.MutableRefObject<HTMLElement | null>;
}

const PopoverContext = React.createContext<PopoverContextValue | undefined>(undefined);

function usePopover() {
  const context = React.useContext(PopoverContext);
  if (!context) throw new Error("usePopover must be used within a Popover");
  return context;
}

export interface PopoverProps {
  children: React.ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

export function Popover({ children, open: controlledOpen, onOpenChange }: PopoverProps) {
  const [uncontrolledOpen, setUncontrolledOpen] = React.useState(false);
  const triggerRef = React.useRef<HTMLElement | null>(null);

  const open = controlledOpen !== undefined ? controlledOpen : uncontrolledOpen;
  const setOpen = React.useCallback(
    (value: boolean) => {
      if (controlledOpen === undefined) setUncontrolledOpen(value);
      onOpenChange?.(value);
    },
    [controlledOpen, onOpenChange]
  );

  return (
    <PopoverContext.Provider value={{ open, setOpen, triggerRef }}>
      <div className="relative inline-block">{children}</div>
    </PopoverContext.Provider>
  );
}

export interface PopoverTriggerProps {
  children: React.ReactNode;
  asChild?: boolean;
}

export function PopoverTrigger({ children, asChild }: PopoverTriggerProps) {
  const { open, setOpen, triggerRef } = usePopover();
  const handleClick = () => setOpen(!open);

  if (asChild && React.isValidElement(children)) {
    const child = children as React.ReactElement<{
      onClick?: (e: React.MouseEvent) => void;
      ref?: React.Ref<HTMLElement>;
    }>;
    return React.cloneElement(child, {
      ref: (node: HTMLElement | null) => {
        triggerRef.current = node;
        const forwardedRef = (child as { ref?: React.Ref<HTMLElement> }).ref;
        if (typeof forwardedRef === "function") forwardedRef(node);
        else if (forwardedRef && typeof forwardedRef === "object") {
          // eslint-disable-next-line react-hooks/immutability
          (forwardedRef as { current: HTMLElement | null }).current = node;
        }
      },
      onClick: (e: React.MouseEvent) => {
        handleClick();
        child.props.onClick?.(e);
      },
    });
  }

  return (
    <button
      type="button"
      ref={(node) => {
        triggerRef.current = node;
      }}
      onClick={handleClick}
    >
      {children}
    </button>
  );
}

export interface PopoverContentProps extends React.HTMLAttributes<HTMLDivElement> {
  align?: "start" | "center" | "end";
  sideOffset?: number;
}

export function PopoverContent({
  children,
  className,
  align = "center",
  sideOffset = 4,
  ...props
}: PopoverContentProps) {
  const { open, setOpen, triggerRef } = usePopover();
  const contentRef = React.useRef<HTMLDivElement>(null);
  const [pos, setPos] = React.useState<{ top: number; left: number } | null>(null);
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  const compute = React.useCallback(() => {
    const trigger = triggerRef.current;
    const content = contentRef.current;
    if (!trigger || !content) return;
    const t = trigger.getBoundingClientRect();
    const w = content.offsetWidth;
    let left: number;
    if (align === "start") left = t.left;
    else if (align === "end") left = t.right - w;
    else left = t.left + t.width / 2 - w / 2;
    const max = window.innerWidth - w - 8;
    if (left < 8) left = 8;
    if (left > max) left = max;
    const top = t.bottom + sideOffset;
    setPos({ top, left });
  }, [align, sideOffset, triggerRef]);

  React.useLayoutEffect(() => {
    if (!open) return;
    compute();
  }, [open, compute]);

  React.useEffect(() => {
    if (!open) return;
    const handler = () => compute();
    window.addEventListener("resize", handler);
    window.addEventListener("scroll", handler, true);
    return () => {
      window.removeEventListener("resize", handler);
      window.removeEventListener("scroll", handler, true);
    };
  }, [open, compute]);

  React.useEffect(() => {
    if (!open) return;
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (contentRef.current?.contains(target)) return;
      if (triggerRef.current?.contains(target)) return;
      setOpen(false);
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [open, setOpen, triggerRef]);

  if (!open || !mounted) return null;

  const content = (
    <div
      ref={contentRef}
      className={cn(
        "fixed z-50 min-w-[8rem] rounded-md border bg-popover p-4 text-popover-foreground shadow-md outline-none",
        "animate-in fade-in-0 zoom-in-95",
        className
      )}
      style={{ top: pos?.top ?? 0, left: pos?.left ?? 0, visibility: pos ? "visible" : "hidden" }}
      {...props}
    >
      {children}
    </div>
  );

  return createPortal(content, document.body);
}
