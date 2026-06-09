"use client";

import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type {
  OneOfCredentialField,
  SimpleCredentialField,
} from "./AgentCredentialsSettings/credentialForms/types";

interface SimpleFieldInputProps {
  field: SimpleCredentialField;
  value: string;
  onChange: (v: string) => void;
  isEditing: boolean;
  isConfigured: boolean;
  isRemoved: boolean;
  onRemove: () => void;
  onRestore: () => void;
  t: (key: string) => string;
}

export function SimpleFieldInput({
  field,
  value,
  onChange,
  isEditing,
  isConfigured,
  isRemoved,
  onRemove,
  onRestore,
  t,
}: SimpleFieldInputProps) {
  // Only an already-stored secret offers a remove button: its value never
  // round-trips, so a blank input means "keep" and an explicit action is the
  // only way to delete it. Non-secret (text) fields clear by blanking instead.
  const removable = isEditing && field.kind === "secret" && isConfigured;

  if (removable && isRemoved) {
    return (
      <div className="grid gap-2">
        <Label>{t(field.label)}</Label>
        <div className="flex items-center justify-between rounded-md border border-dashed border-destructive/50 px-3 py-2 text-sm text-muted-foreground">
          <span>{t("settings.agentCredentials.willRemove")}</span>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            data-testid={`restore-secret-${field.envKey}`}
            onClick={onRestore}
          >
            {t("common.undo")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="grid gap-2">
      <Label htmlFor={`cred-${field.envKey}`}>{t(field.label)}</Label>
      {field.description && (
        <p className="text-xs text-muted-foreground">{t(field.description)}</p>
      )}
      <div className="flex items-center gap-2">
        <Input
          id={`cred-${field.envKey}`}
          type={field.kind === "secret" ? "password" : "text"}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="flex-1"
          placeholder={
            isEditing && field.kind === "secret"
              ? t("settings.agentCredentials.secretPlaceholder")
              : field.placeholder ?? ""
          }
        />
        {removable && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            data-testid={`remove-secret-${field.envKey}`}
            onClick={onRemove}
            title={t("settings.agentCredentials.removeSecret")}
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        )}
      </div>
      {field.securityHint && (
        <p className="text-xs text-amber-600 dark:text-amber-500">
          {t(field.securityHint)}
        </p>
      )}
      {isEditing && field.kind === "secret" && (
        <p className="text-xs text-muted-foreground">
          {t("settings.agentCredentials.secretEditHint")}
        </p>
      )}
    </div>
  );
}

interface OneOfFieldInputProps {
  field: OneOfCredentialField;
  selectedEnvKey: string;
  onSelect: (envKey: string) => void;
  values: Record<string, string>;
  onValueChange: (envKey: string, value: string) => void;
  isEditing: boolean;
  t: (key: string) => string;
}

export function OneOfFieldInput({
  field,
  selectedEnvKey,
  onSelect,
  values,
  onValueChange,
  isEditing,
  t,
}: OneOfFieldInputProps) {
  const selected =
    field.options.find((o) => o.envKey === selectedEnvKey) ?? field.options[0];
  return (
    <div className="grid gap-2">
      <Label>{t(field.label)}</Label>
      {field.description && (
        <p className="text-xs text-muted-foreground">{t(field.description)}</p>
      )}
      <div
        role="radiogroup"
        aria-label={t(field.label)}
        className="flex flex-wrap gap-2"
      >
        {field.options.map((opt) => {
          const isActive = opt.envKey === selected.envKey;
          return (
            <label
              key={opt.envKey}
              data-testid={`oneof-option-${field.group}-${opt.envKey}`}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-md border cursor-pointer text-sm transition-colors ${
                isActive
                  ? "border-primary bg-primary/10 text-foreground"
                  : "border-border hover:bg-muted/50"
              }`}
            >
              <input
                type="radio"
                name={field.group}
                value={opt.envKey}
                checked={isActive}
                onChange={() => onSelect(opt.envKey)}
                className="accent-primary"
              />
              {t(opt.label)}
            </label>
          );
        })}
      </div>
      <Input
        id={`cred-${selected.envKey}`}
        type={selected.kind === "secret" ? "password" : "text"}
        value={values[selected.envKey] ?? ""}
        onChange={(e) => onValueChange(selected.envKey, e.target.value)}
        placeholder={
          isEditing && selected.kind === "secret"
            ? t("settings.agentCredentials.secretPlaceholder")
            : selected.placeholder ?? ""
        }
      />
      {isEditing && selected.kind === "secret" && (
        <p className="text-xs text-muted-foreground">
          {t("settings.agentCredentials.secretEditHint")}
        </p>
      )}
    </div>
  );
}
