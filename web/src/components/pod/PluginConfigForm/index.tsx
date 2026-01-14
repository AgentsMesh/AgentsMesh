"use client";

import React, { memo, useCallback } from "react";
import type { PluginCapability, UIField } from "@/lib/api/runner";
import { getFieldRenderer } from "./field-renderers";

interface PluginConfigFormProps {
  plugins: PluginCapability[];
  values: Record<string, unknown>;
  onChange: (pluginName: string, fieldName: string, value: unknown) => void;
}

interface FieldWrapperProps {
  plugin: PluginCapability;
  field: UIField;
  value: unknown;
  onChange: (pluginName: string, fieldName: string, value: unknown) => void;
}

/**
 * Wrapper component for individual fields
 * Handles the field key generation and change propagation
 */
const FieldWrapper = memo(function FieldWrapper({
  plugin,
  field,
  value,
  onChange,
}: FieldWrapperProps) {
  const fieldKey = `${plugin.name}.${field.name}`;

  // Get the renderer component - stable reference from the registry
  const Renderer = getFieldRenderer(field.type);

  const handleChange = useCallback(
    (newValue: unknown) => {
      onChange(plugin.name, field.name, newValue);
    },
    [plugin.name, field.name, onChange]
  );

  // Use createElement to avoid React Compiler's component creation detection
  return React.createElement(Renderer, {
    fieldKey,
    field,
    value,
    onChange: handleChange,
  });
});

/**
 * Dynamic form renderer for plugin configuration.
 * Uses strategy pattern for field rendering - each field type has its own renderer.
 *
 * To add a new field type:
 * 1. Create a new renderer component in field-renderers.tsx
 * 2. Register it in the FIELD_RENDERERS map
 */
export const PluginConfigForm = memo(function PluginConfigForm({
  plugins,
  values,
  onChange,
}: PluginConfigFormProps) {
  if (!plugins || plugins.length === 0) {
    return null;
  }

  return (
    <div className="space-y-4">
      {plugins.map((plugin) => (
        <div key={plugin.name} className="border border-border rounded-md p-3">
          <h4 className="text-sm font-medium mb-3">
            {plugin.description || plugin.name}
          </h4>
          <div className="space-y-3">
            {plugin.ui?.fields.map((field) => {
              const fieldKey = `${plugin.name}.${field.name}`;
              const currentValue = values[fieldKey] ?? field.default;

              return (
                <FieldWrapper
                  key={fieldKey}
                  plugin={plugin}
                  field={field}
                  value={currentValue}
                  onChange={onChange}
                />
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
});

// Re-export types for external use
export type { FieldRenderer } from "./field-renderers";

export default PluginConfigForm;
