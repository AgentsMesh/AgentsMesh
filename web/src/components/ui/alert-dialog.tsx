"use client";

import { useEffect, useRef, useCallback, createContext, useContext } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";
import { Button } from "./button";

// Context to propagate onOpenChange to Cancel/Action children
const AlertDialogContext = createContext<{
  onOpenChange: (open: boolean) => void;
}>({ onOpenChange: () => {} });

interface AlertDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

export function AlertDialog({ open, onOpenChange, children }: AlertDialogProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, onOpenChange]);

  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => { document.body.style.overflow = ""; };
  }, [open]);

  if (!open) return null;

  return createPortal(
    <AlertDialogContext.Provider value={{ onOpenChange }}>
      <div
        ref={overlayRef}
        className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
      >
        {children}
      </div>
    </AlertDialogContext.Provider>,
    document.body
  );
}

export function AlertDialogContent({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "bg-background rounded-lg shadow-lg w-full max-w-md p-6",
        className
      )}
      onClick={(e) => e.stopPropagation()}
    >
      {children}
    </div>
  );
}

export function AlertDialogHeader({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("space-y-2", className)}>{children}</div>;
}

export function AlertDialogTitle({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <h2 className={cn("text-lg font-semibold", className)}>{children}</h2>;
}

export function AlertDialogDescription({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <p className={cn("text-sm text-muted-foreground", className)}>{children}</p>
  );
}

export function AlertDialogFooter({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("mt-4 flex items-center justify-end gap-2", className)}>
      {children}
    </div>
  );
}

export function AlertDialogCancel({
  children,
  className,
  onClick,
}: {
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
}) {
  const { onOpenChange } = useContext(AlertDialogContext);

  const handleClick = useCallback(() => {
    onClick?.();
    onOpenChange(false);
  }, [onClick, onOpenChange]);

  return (
    <Button variant="outline" className={className} onClick={handleClick}>
      {children}
    </Button>
  );
}

export function AlertDialogAction({
  children,
  className,
  onClick,
}: {
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
}) {
  return (
    <Button className={className} onClick={onClick}>
      {children}
    </Button>
  );
}
