"use client";

import React from "react";

interface BaseFieldProps {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
}

function FieldDescription({ fieldKey, description }: { fieldKey: string; description: string }) {
  if (!description) return null;
  return <p id={`${fieldKey}-desc`} className="text-xs text-muted-foreground mt-1">{description}</p>;
}

export function BooleanField({ fieldKey, label, description, value, onChange }: BaseFieldProps) {
  return (
    <div className="flex items-center gap-2">
      <input type="checkbox" id={fieldKey} checked={Boolean(value)}
        onChange={(e) => onChange(e.target.checked)} className="h-4 w-4 rounded border-border"
        aria-describedby={description ? `${fieldKey}-desc` : undefined} />
      <label htmlFor={fieldKey} className="text-sm">{label}</label>
      {description && (
        <span id={`${fieldKey}-desc`} className="text-xs text-muted-foreground ml-auto">{description}</span>
      )}
    </div>
  );
}

export function StringField({ fieldKey, label, description, value, onChange, required }: BaseFieldProps) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}{required && <span className="text-destructive ml-1">*</span>}
      </label>
      <input type="text" id={fieldKey} value={String(value ?? "")}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined} aria-required={required} />
      <FieldDescription fieldKey={fieldKey} description={description} />
    </div>
  );
}

export function SecretField({ fieldKey, label, description, value, onChange, required }: BaseFieldProps) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}{required && <span className="text-destructive ml-1">*</span>}
      </label>
      <input type="password" id={fieldKey} value={String(value ?? "")}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined} aria-required={required} />
      <FieldDescription fieldKey={fieldKey} description={description} />
    </div>
  );
}

export function NumberField({ fieldKey, label, description, value, onChange, required, min, max }: BaseFieldProps & {
  min?: number; max?: number;
}) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}{required && <span className="text-destructive ml-1">*</span>}
      </label>
      <input type="number" id={fieldKey} value={value != null ? Number(value) : ""} min={min} max={max}
        onChange={(e) => onChange(e.target.value ? Number(e.target.value) : null)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined} aria-required={required} />
      <FieldDescription fieldKey={fieldKey} description={description} />
    </div>
  );
}

export function SelectField({ fieldKey, label, description, value, onChange, required, options, getOptionLabel }: BaseFieldProps & {
  options?: { value: string }[]; getOptionLabel: (value: string) => string;
}) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}{required && <span className="text-destructive ml-1">*</span>}
      </label>
      <select id={fieldKey} value={String(value ?? "")} onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined} aria-required={required}>
        {!required && !value && !options?.some((o) => o.value === "") && (
          <option value="" disabled>Select {label.toLowerCase()}...</option>
        )}
        {options?.map((option) => (
          <option key={option.value} value={option.value}>{getOptionLabel(option.value)}</option>
        ))}
      </select>
      <FieldDescription fieldKey={fieldKey} description={description} />
    </div>
  );
}
