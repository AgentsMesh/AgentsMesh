"use client";

import { useEffect, useRef, useCallback, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

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
}

interface ResponsiveDialogHeaderProps {
  children: React.ReactNode;
  className?: string;
  /** Close handler for mobile back/close button */
  onClose?: () => void;
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
 * - Both mobile and desktop: floating centered modal dialog with overlay
 * - Uses responsive padding (px-4 on mobile, px-6 on desktop)
 */
export function ResponsiveDialog({
  open,
  onOpenChange,
  children,
}: ResponsiveDialogProps) {
  const overlayRef = useRef<HTMLDivElement>(null);
  const mounted = useIsMounted();

  // Handle escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, onOpenChange]);

  // Prevent body scroll when open
  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  const handleOverlayClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === overlayRef.current) {
        onOpenChange(false);
      }
    },
    [onOpenChange]
  );

  // Wait for mount to ensure document.body is available (SSR-safe)
  if (!open || !mounted) return null;

  // Both mobile and desktop: centered floating dialog with overlay
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
}: ResponsiveDialogContentProps) {
  return (
    <div
      className={cn(
        "bg-background border border-border rounded-lg shadow-lg w-full max-w-lg max-h-[90vh] overflow-y-auto p-4 md:p-6",
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
  onClose,
}: ResponsiveDialogHeaderProps) {
  return (
    <div className={cn("flex items-center justify-between mb-4", className)}>
      {children}
      {onClose && (
        <button
          onClick={onClose}
          className="rounded-sm opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring ml-2"
        >
          <X className="h-5 w-5" />
          <span className="sr-only">Close</span>
        </button>
      )}
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
  return (
    <div className={cn(className)}>
      {children}
    </div>
  );
}

export function ResponsiveDialogFooter({
  children,
  className,
}: ResponsiveDialogFooterProps) {
  return (
    <div className={cn("flex items-center gap-2 mt-4 flex-col-reverse md:flex-row md:justify-end", className)}>
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
