"use client";

import { useEffect, useRef, useCallback, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import { Drawer } from "vaul";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useBreakpoint } from "@/components/layout/useBreakpoint";

// SSR-safe hook to detect client-side mounting
const emptySubscribe = () => () => {};
function useIsMounted() {
  return useSyncExternalStore(
    emptySubscribe,
    () => true,  // Client: always mounted
    () => false  // Server: never mounted
  );
}

interface ResponsiveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

interface ResponsiveDialogContentProps {
  children: React.ReactNode;
  className?: string;
  /** Title for accessibility (shown in drawer handle area on mobile) */
  title?: string;
}

interface ResponsiveDialogHeaderProps {
  children: React.ReactNode;
  className?: string;
}

interface ResponsiveDialogTitleProps {
  children: React.ReactNode;
  className?: string;
}

interface ResponsiveDialogDescriptionProps {
  children: React.ReactNode;
  className?: string;
}

interface ResponsiveDialogBodyProps {
  children: React.ReactNode;
  className?: string;
}

interface ResponsiveDialogFooterProps {
  children: React.ReactNode;
  className?: string;
}

interface ResponsiveDialogCloseProps {
  onClose: () => void;
  className?: string;
}

/**
 * ResponsiveDialog - A dialog component that adapts to screen size
 * - Desktop: Shows as a centered modal dialog
 * - Mobile: Shows as a bottom drawer (vaul)
 */
export function ResponsiveDialog({
  open,
  onOpenChange,
  children,
}: ResponsiveDialogProps) {
  const { isMobile } = useBreakpoint();
  const overlayRef = useRef<HTMLDivElement>(null);
  const mounted = useIsMounted();

  // Handle escape key (for desktop mode)
  useEffect(() => {
    if (isMobile) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, onOpenChange, isMobile]);

  // Prevent body scroll when open (for desktop mode)
  useEffect(() => {
    if (isMobile) return;

    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open, isMobile]);

  const handleOverlayClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === overlayRef.current) {
        onOpenChange(false);
      }
    },
    [onOpenChange]
  );

  // Mobile: Use vaul Drawer
  if (isMobile) {
    return (
      <Drawer.Root open={open} onOpenChange={onOpenChange}>
        <Drawer.Portal>
          <Drawer.Overlay className="fixed inset-0 bg-black/40 z-50" />
          {children}
        </Drawer.Portal>
      </Drawer.Root>
    );
  }

  // Desktop: Use standard dialog with Portal
  // Wait for mount to ensure document.body is available (SSR-safe)
  if (!open || !mounted) return null;

  // Use Portal to render dialog at document body level
  return createPortal(
    <div
      ref={overlayRef}
      className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
      onClick={handleOverlayClick}
    >
      {children}
    </div>,
    document.body
  );
}

export function ResponsiveDialogContent({
  children,
  className,
  title,
}: ResponsiveDialogContentProps) {
  const { isMobile } = useBreakpoint();

  // Mobile: Drawer content
  if (isMobile) {
    return (
      <Drawer.Content
        className={cn(
          "fixed bottom-0 left-0 right-0 bg-background rounded-t-2xl z-50 max-h-[85dvh] flex flex-col",
          className
        )}
        aria-describedby={undefined}
      >
        {/* Handle */}
        <div className="flex justify-center pt-3 pb-2 flex-shrink-0">
          <div className="w-10 h-1 rounded-full bg-muted" />
        </div>
        {/* Hidden title for accessibility */}
        {title && <Drawer.Title className="sr-only">{title}</Drawer.Title>}
        <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
          {children}
        </div>
        {/* Safe area padding */}
        <div className="h-safe flex-shrink-0" />
      </Drawer.Content>
    );
  }

  // Desktop: Dialog content
  return (
    <div
      className={cn(
        "bg-background rounded-lg shadow-lg w-full max-w-lg max-h-[90vh] overflow-hidden flex flex-col",
        className
      )}
      onClick={(e) => e.stopPropagation()}
    >
      {children}
    </div>
  );
}

export function ResponsiveDialogHeader({
  children,
  className,
}: ResponsiveDialogHeaderProps) {
  const { isMobile } = useBreakpoint();

  return (
    <div
      className={cn(
        "px-6 py-4 border-b flex-shrink-0",
        isMobile && "px-4",
        className
      )}
    >
      {children}
    </div>
  );
}

export function ResponsiveDialogTitle({
  children,
  className,
}: ResponsiveDialogTitleProps) {
  return (
    <h2 className={cn("text-lg font-semibold", className)}>{children}</h2>
  );
}

export function ResponsiveDialogDescription({
  children,
  className,
}: ResponsiveDialogDescriptionProps) {
  return (
    <p className={cn("text-sm text-muted-foreground mt-1", className)}>
      {children}
    </p>
  );
}

export function ResponsiveDialogBody({
  children,
  className,
}: ResponsiveDialogBodyProps) {
  const { isMobile } = useBreakpoint();

  return (
    <div
      className={cn(
        "px-6 py-4 flex-1 overflow-y-auto",
        isMobile && "px-4 overscroll-contain",
        className
      )}
    >
      {children}
    </div>
  );
}

export function ResponsiveDialogFooter({
  children,
  className,
}: ResponsiveDialogFooterProps) {
  const { isMobile } = useBreakpoint();

  return (
    <div
      className={cn(
        "px-6 py-4 border-t flex items-center gap-2 flex-shrink-0",
        isMobile ? "px-4 flex-col-reverse" : "justify-end",
        className
      )}
    >
      {children}
    </div>
  );
}

export function ResponsiveDialogClose({
  onClose,
  className,
}: ResponsiveDialogCloseProps) {
  return (
    <button
      onClick={onClose}
      className={cn(
        "absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
        className
      )}
    >
      <X className="h-4 w-4" />
      <span className="sr-only">Close</span>
    </button>
  );
}
