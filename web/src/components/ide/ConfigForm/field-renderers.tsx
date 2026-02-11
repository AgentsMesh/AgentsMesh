"use client";

import React, { memo } from "react";
import type { ConfigField } from "@/lib/api/agent";
import { useTranslations } from "next-intl";

/**
 * Props for field renderer components
 */
export interface FieldRendererProps {
  fieldKey: string;
  field: ConfigField;
  value: unknown;
  onChange: (value: unknown) => void;
  /** Agent slug for i18n translation key construction */
  agentSlug: string;
}

/**
 * Hook for getting translated field labels and descriptions
 * Uses the pattern: agent.{agentSlug}.fields.{fieldName}.label/description
 */
function useFieldTranslation(agentSlug: string, fieldName: string) {
  const t = useTranslations();
  const basePath = `agent.${agentSlug}.fields.${fieldName}`;

  return {
    label: t(`${basePath}.label`),
    description: t(`${basePath}.description`),
    getOptionLabel: (optionValue: string) => {
      // For empty string option values, use a special key
      const key = optionValue === "" ? `${basePath}.options.` : `${basePath}.options.${optionValue}`;
      return t(key);
    },
  };
}

/**
 * Boolean field renderer (checkbox)
 */
function BooleanField({
  fieldKey,
  label,
  description,
  value,
  onChange,
}: {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <input
        type="checkbox"
        id={fieldKey}
        checked={Boolean(value)}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 rounded border-border"
        aria-describedby={description ? `${fieldKey}-desc` : undefined}
      />
      <label htmlFor={fieldKey} className="text-sm">
        {label}
      </label>
      {description && (
        <span id={`${fieldKey}-desc`} className="text-xs text-muted-foreground ml-auto">
          {description}
        </span>
      )}
    </div>
  );
}

/**
 * String field renderer (text input)
 */
function StringField({
  fieldKey,
  label,
  description,
  value,
  onChange,
  required,
}: {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
}) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </label>
      <input
        type="text"
        id={fieldKey}
        value={String(value ?? "")}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined}
        aria-required={required}
      />
      {description && (
        <p id={`${fieldKey}-desc`} className="text-xs text-muted-foreground mt-1">
          {description}
        </p>
      )}
    </div>
  );
}

/**
 * Secret field renderer (password input)
 */
function SecretField({
  fieldKey,
  label,
  description,
  value,
  onChange,
  required,
}: {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
}) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </label>
      <input
        type="password"
        id={fieldKey}
        value={String(value ?? "")}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined}
        aria-required={required}
      />
      {description && (
        <p id={`${fieldKey}-desc`} className="text-xs text-muted-foreground mt-1">
          {description}
        </p>
      )}
    </div>
  );
}

/**
 * Number field renderer
 */
function NumberField({
  fieldKey,
  label,
  description,
  value,
  onChange,
  required,
  min,
  max,
}: {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
  min?: number;
  max?: number;
}) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </label>
      <input
        type="number"
        id={fieldKey}
        value={value != null ? Number(value) : ""}
        min={min}
        max={max}
        onChange={(e) => onChange(e.target.value ? Number(e.target.value) : null)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined}
        aria-required={required}
      />
      {description && (
        <p id={`${fieldKey}-desc`} className="text-xs text-muted-foreground mt-1">
          {description}
        </p>
      )}
    </div>
  );
}

/**
 * Select field renderer (dropdown)
 */
function SelectField({
  fieldKey,
  label,
  description,
  value,
  onChange,
  required,
  options,
  getOptionLabel,
}: {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
  options?: { value: string }[];
  getOptionLabel: (value: string) => string;
}) {
  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </label>
      <select
        id={fieldKey}
        value={String(value ?? "")}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
        aria-describedby={description ? `${fieldKey}-desc` : undefined}
        aria-required={required}
      >
        {!required && !value && (
          <option value="" disabled>
            Select {label.toLowerCase()}...
          </option>
        )}
        {options?.map((option) => (
          <option key={option.value} value={option.value}>
            {getOptionLabel(option.value)}
          </option>
        ))}
      </select>
      {description && (
        <p id={`${fieldKey}-desc`} className="text-xs text-muted-foreground mt-1">
          {description}
        </p>
      )}
    </div>
  );
}

/**
 * Unified field renderer component
 * Uses switch statement internally to select the appropriate field type rendering
 * This pattern avoids dynamic component creation during render (react-compiler compliant)
 */
export const FieldRenderer = memo(function FieldRenderer({
  fieldKey,
  field,
  value,
  onChange,
  agentSlug,
}: FieldRendererProps) {
  const { label, description, getOptionLabel } = useFieldTranslation(agentSlug, field.name);

  switch (field.type) {
    case "boolean":
      return (
        <BooleanField
          fieldKey={fieldKey}
          label={label}
          description={description}
          value={value}
          onChange={onChange}
        />
      );

    case "string":
      return (
        <StringField
          fieldKey={fieldKey}
          label={label}
          description={description}
          value={value}
          onChange={onChange}
          required={field.required}
        />
      );

    case "secret":
      return (
        <SecretField
          fieldKey={fieldKey}
          label={label}
          description={description}
          value={value}
          onChange={onChange}
          required={field.required}
        />
      );

    case "number":
      return (
        <NumberField
          fieldKey={fieldKey}
          label={label}
          description={description}
          value={value}
          onChange={onChange}
          required={field.required}
          min={field.validation?.min}
          max={field.validation?.max}
        />
      );

    case "select":
      return (
        <SelectField
          fieldKey={fieldKey}
          label={label}
          description={description}
          value={value}
          onChange={onChange}
          required={field.required}
          options={field.options}
          getOptionLabel={getOptionLabel}
        />
      );

    default:
      return (
        <div className="text-sm text-muted-foreground">
          Unknown field type: {field.type} ({fieldKey})
        </div>
      );
  }
});
