"use client";

import React, { memo } from "react";
import type { ConfigField } from "@/lib/api/agent";
import { useTranslations } from "next-intl";
import {
  BooleanField, StringField, SecretField, NumberField, SelectField,
} from "./field-components";

export interface FieldRendererProps {
  fieldKey: string;
  field: ConfigField;
  value: unknown;
  onChange: (value: unknown) => void;
  agentSlug: string;
  values?: Record<string, unknown>;
}

function useFieldTranslation(agentSlug: string, fieldName: string) {
  const t = useTranslations();
  const basePath = `agent.${agentSlug}.fields.${fieldName}`;
  const humanized = fieldName.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

  return {
    label: t.has(`${basePath}.label`) ? t(`${basePath}.label`) : humanized,
    description: t.has(`${basePath}.description`) ? t(`${basePath}.description`) : "",
    getOptionLabel: (optionValue: string) => {
      const key = optionValue === "" ? `${basePath}.options.` : `${basePath}.options.${optionValue}`;
      return t.has(key) ? t(key) : optionValue || humanized;
    },
  };
}

export const FieldRenderer = memo(function FieldRenderer({
  fieldKey, field, value, onChange, agentSlug,
}: FieldRendererProps) {
  const { label, description, getOptionLabel } = useFieldTranslation(agentSlug, field.name);
  const common = { fieldKey, label, description, value, onChange, required: field.required };

  switch (field.type) {
    case "boolean":
      return <BooleanField {...common} />;
    case "string":
      return <StringField {...common} />;
    case "secret":
      return <SecretField {...common} />;
    case "number":
      return <NumberField {...common} min={field.validation?.min} max={field.validation?.max} />;
    case "select":
      return <SelectField {...common} options={field.options} getOptionLabel={getOptionLabel} />;
    default:
      return (
        <div className="text-sm text-muted-foreground">
          Unknown field type: {field.type} ({fieldKey})
        </div>
      );
  }
});
