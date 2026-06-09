"use client";

import { useMemo } from "react";
import type {
  CredentialFormSpec,
  CredentialFieldSpec,
  CustomEnvEntry,
} from "./AgentCredentialsSettings/credentialForms/types";
import { getEnvKeysFromSpec } from "./AgentCredentialsSettings/credentialForms";
import { CustomEnvSection } from "./CustomEnvSection";
import { SimpleFieldInput, OneOfFieldInput } from "./credentialFieldInputs";

interface CredentialFormFieldsProps {
  spec: CredentialFormSpec;
  values: Record<string, string>;
  onValueChange: (envKey: string, value: string) => void;
  selectedOneOf: Record<string, string>;
  onOneOfChange: (group: string, envKey: string) => void;
  customEnv: CustomEnvEntry[];
  onCustomEnvChange: (entries: CustomEnvEntry[]) => void;
  // Editing-only: configuredKeys gates the per-secret remove button (only a
  // stored secret can be deleted) and the remove/restore callbacks toggle
  // removedKeys. Optional so the create path and unit tests need not thread them.
  configuredKeys?: string[];
  removedKeys?: string[];
  onRemoveKey?: (envKey: string) => void;
  onRestoreKey?: (envKey: string) => void;
  isEditing: boolean;
  t: (key: string) => string;
}

function renderField(
  field: CredentialFieldSpec,
  props: CredentialFormFieldsProps,
  configured: Set<string>,
  removed: Set<string>
): React.ReactNode {
  if (field.kind === "oneof") {
    return (
      <OneOfFieldInput
        key={field.group}
        field={field}
        selectedEnvKey={props.selectedOneOf[field.group] ?? field.options[0]?.envKey ?? ""}
        onSelect={(envKey) => props.onOneOfChange(field.group, envKey)}
        values={props.values}
        onValueChange={props.onValueChange}
        isEditing={props.isEditing}
        t={props.t}
      />
    );
  }
  return (
    <SimpleFieldInput
      key={field.envKey}
      field={field}
      value={props.values[field.envKey] ?? ""}
      onChange={(v) => props.onValueChange(field.envKey, v)}
      isEditing={props.isEditing}
      isConfigured={configured.has(field.envKey)}
      isRemoved={removed.has(field.envKey)}
      onRemove={() => props.onRemoveKey?.(field.envKey)}
      onRestore={() => props.onRestoreKey?.(field.envKey)}
      t={props.t}
    />
  );
}

export function CredentialFormFields(props: CredentialFormFieldsProps) {
  const { spec, customEnv, onCustomEnvChange, configuredKeys = [], removedKeys = [], isEditing, t } =
    props;
  const declaredKeys = useMemo(() => getEnvKeysFromSpec(spec), [spec]);
  const configured = useMemo(() => new Set(configuredKeys), [configuredKeys]);
  const removed = useMemo(() => new Set(removedKeys), [removedKeys]);

  if (spec.fields.length === 0 && !spec.allowCustomEnv) return null;

  return (
    <>
      {spec.fields.map((field) => renderField(field, props, configured, removed))}
      {spec.allowCustomEnv && (
        <CustomEnvSection
          title={t("settings.credentialForm.customEnv.title")}
          hint={spec.customEnvHint ? t(spec.customEnvHint) : undefined}
          entries={customEnv}
          declaredKeys={declaredKeys}
          onChange={onCustomEnvChange}
          isEditing={isEditing}
          t={t}
        />
      )}
    </>
  );
}

export default CredentialFormFields;
