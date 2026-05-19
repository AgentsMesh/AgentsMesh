"use client";

import React, { memo, useCallback } from "react";
import type { ConfigField } from "@/lib/api";
import { FieldRenderer } from "./field-renderers";

interface ConfigFormProps {
  fields: ConfigField[];
  values: Record<string, unknown>;
  onChange: (fieldName: string, value: unknown) => void;
  agentSlug: string;
}

interface FieldWrapperProps {
  field: ConfigField;
  value: unknown;
  onChange: (fieldName: string, value: unknown) => void;
  agentSlug: string;
}

const FieldWrapper = memo(function FieldWrapper({
  field,
  value,
  onChange,
  agentSlug,
  values,
}: FieldWrapperProps & { values?: Record<string, unknown> }) {
  const fieldKey = field.name;

  const handleChange = useCallback(
    (newValue: unknown) => {
      onChange(field.name, newValue);
    },
    [field.name, onChange]
  );

  return (
    <FieldRenderer
      fieldKey={fieldKey}
      field={field}
      value={value}
      onChange={handleChange}
      agentSlug={agentSlug}
      values={values}
    />
  );
});

/**
 * Dynamic form renderer for agent configuration.
 * Uses switch-based rendering for field types (react-compiler compliant).
 *
 * i18n: Labels and descriptions are translated on frontend using:
 * - agent.{agentSlug}.fields.{field.name}.label
 * - agent.{agentSlug}.fields.{field.name}.description
 * - agent.{agentSlug}.fields.{field.name}.options.{optionValue}
 *
 * To add a new field type:
 * 1. Add a new case in the FieldRenderer switch statement in field-renderers.tsx
 * 2. Create the corresponding field component if needed
 */
export const ConfigForm = memo(function ConfigForm({
  fields,
  values,
  onChange,
  agentSlug,
}: ConfigFormProps) {
  if (!fields || fields.length === 0) {
    return null;
  }

  return (
    <div className="space-y-4">
      <div className="border border-border rounded-md p-3">
        <div className="space-y-3">
          {fields.map((field) => {
            const currentValue = values[field.name] ?? field.default;

            return (
              <FieldWrapper
                key={field.name}
                field={field}
                value={currentValue}
                onChange={onChange}
                agentSlug={agentSlug}
                values={values}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
});

export type { FieldRendererProps } from "./field-renderers";

export default ConfigForm;
