"use client";

import * as React from "react";
import { Label } from "./label";
import { cn } from "@/lib/utils";

export interface FormFieldProps {
  label: string;
  htmlFor?: string;
  error?: string;
  hint?: string;
  required?: boolean;
  disabled?: boolean;
  className?: string;
  children: React.ReactNode;
}

export function FormField({
  label,
  htmlFor,
  error,
  hint,
  required,
  disabled,
  className,
  children,
}: FormFieldProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <Label
        htmlFor={htmlFor}
        className={cn(
          "block text-sm font-medium",
          disabled && "opacity-50 cursor-not-allowed"
        )}
      >
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </Label>
      {children}
      {error && (
        <p className="text-xs text-destructive" role="alert">
          {error}
        </p>
      )}
      {hint && !error && (
        <p className="text-xs text-muted-foreground">{hint}</p>
      )}
    </div>
  );
}

export interface FormFieldGroupProps {
  title?: string;
  description?: string;
  className?: string;
  children: React.ReactNode;
}

export function FormFieldGroup({
  title,
  description,
  className,
  children,
}: FormFieldGroupProps) {
  return (
    <div className={cn("space-y-4", className)}>
      {(title || description) && (
        <div className="space-y-1">
          {title && <h3 className="text-sm font-medium">{title}</h3>}
          {description && (
            <p className="text-sm text-muted-foreground">{description}</p>
          )}
        </div>
      )}
      <div className="space-y-4">{children}</div>
    </div>
  );
}

export interface FormRowProps {
  className?: string;
  gap?: 2 | 3 | 4 | 6 | 8;
  children: React.ReactNode;
}

export function FormRow({ className, gap = 4, children }: FormRowProps) {
  const gapClass = {
    2: "gap-2",
    3: "gap-3",
    4: "gap-4",
    6: "gap-6",
    8: "gap-8",
  }[gap];

  return (
    <div className={cn("flex flex-col sm:flex-row", gapClass, className)}>
      {children}
    </div>
  );
}
