"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { hasInvalidCustomEnvKey } from "./CustomEnvSection";
import type { CredentialProfileViewModel } from "./_shared/credentialViewModel";
import { getConfiguredKeys } from "./_shared/credentialViewModel";
import {
  getCredentialFormSpec,
  getEnvKeysFromSpec,
} from "./AgentCredentialsSettings/credentialForms";
import {
  initFormStateFromProfile,
  buildCredentialsPayload,
  type CredentialFormState,
} from "./AgentCredentialsSettings/credentialForms/formState";
import type { CredentialFormData } from "./AgentCredentialsSettings/credentialForms/types";

export interface CredentialFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentSlug: string;
  editingProfile: CredentialProfileViewModel | null;
  onSubmit: (
    data: CredentialFormData,
    editingProfile: CredentialProfileViewModel | null
  ) => Promise<void>;
  t: (key: string) => string;
}

export function useCredentialDialogForm({
  open,
  agentSlug,
  editingProfile,
  onSubmit,
  onOpenChange,
  t,
}: CredentialFormDialogProps) {
  const spec = useMemo(() => getCredentialFormSpec(agentSlug), [agentSlug]);
  const declaredKeys = useMemo(() => getEnvKeysFromSpec(spec), [spec]);
  const previousKeys = useMemo(
    () => (editingProfile ? getConfiguredKeys(editingProfile) : []),
    [editingProfile]
  );

  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formState, setFormState] = useState<CredentialFormState>(() =>
    initFormStateFromProfile(spec, null)
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setFormName(editingProfile?.name ?? "");
    setFormDescription(editingProfile?.description ?? "");
    setFormState(initFormStateFromProfile(spec, editingProfile));
    setError(null);
  }, [open, editingProfile, spec]);

  const onValueChange = useCallback((envKey: string, value: string) => {
    setFormState((prev) => ({ ...prev, values: { ...prev.values, [envKey]: value } }));
  }, []);

  const onOneOfChange = useCallback((group: string, envKey: string) => {
    setFormState((prev) => ({
      ...prev,
      selectedOneOf: { ...prev.selectedOneOf, [group]: envKey },
    }));
  }, []);

  const onCustomEnvChange = useCallback((entries: CredentialFormState["customEnv"]) => {
    setFormState((prev) => ({ ...prev, customEnv: entries }));
  }, []);

  const onRemoveKey = useCallback((envKey: string) => {
    // A removable field is always a stored secret whose value never prefills, so
    // there is no input value to clear — just mark it removed.
    // buildCredentialsPayload emits "" for a removedKeys entry; leaving values
    // untouched means onRestoreKey (which only drops the entry) fully reverses it.
    setFormState((prev) => ({
      ...prev,
      removedKeys: prev.removedKeys.includes(envKey)
        ? prev.removedKeys
        : [...prev.removedKeys, envKey],
    }));
  }, []);

  const onRestoreKey = useCallback((envKey: string) => {
    setFormState((prev) => ({
      ...prev,
      removedKeys: prev.removedKeys.filter((k) => k !== envKey),
    }));
  }, []);

  const customEnvInvalid = hasInvalidCustomEnvKey(formState.customEnv, declaredKeys);

  const handleSubmit = async () => {
    if (!formName.trim() || customEnvInvalid) return;
    try {
      setSubmitting(true);
      setError(null);
      const credentials = buildCredentialsPayload(spec, formState, previousKeys);
      await onSubmit(
        { name: formName, description: formDescription, credentials },
        editingProfile
      );
      onOpenChange(false);
    } catch (err) {
      console.error("Failed to save profile:", err);
      setError(t("settings.agentCredentials.failedToSave"));
    } finally {
      setSubmitting(false);
    }
  };

  return {
    spec,
    previousKeys,
    formName,
    setFormName,
    formDescription,
    setFormDescription,
    formState,
    submitting,
    error,
    customEnvInvalid,
    onValueChange,
    onOneOfChange,
    onCustomEnvChange,
    onRemoveKey,
    onRestoreKey,
    handleSubmit,
  };
}
