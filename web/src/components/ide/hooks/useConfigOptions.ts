import { useState, useEffect, useCallback, useMemo } from "react";
import { agentApi, userAgentConfigApi, ConfigField } from "@/lib/api";

export interface ConfigOptionsState {
  fields: ConfigField[];
  loading: boolean;
  config: Record<string, unknown>;
  updateConfig: (fieldName: string, value: unknown) => void;
  resetConfig: () => void;
}

/**
 * Derives a `model` select field from a `models` list field.
 * Returns null if no models list exists.
 */
function deriveModelField(models: unknown): ConfigField | null {
  const modelList = Array.isArray(models) ? models.filter((m): m is string => typeof m === "string") : [];
  if (modelList.length === 0) {
    return null;
  }
  return {
    name: "model",
    type: "select",
    default: "",
    options: modelList.map((m) => ({ value: m })),
  };
}

/**
 * Hook to manage agent config options and configuration
 * Loads config schema from Backend when agent is selected
 *
 * Configuration priority (high to low):
 * 1. User overrides in the form
 * 2. User personal config (from personal settings)
 * 3. Backend ConfigSchema defaults
 *
 * Special handling for model_list fields: derives a `model` select
 * field from the models list for runtime selection (not persisted to user config).
 */
export function useConfigOptions(
  runnerId: number | null,
  agentSlug?: string | null
): ConfigOptionsState {
  const [fields, setFields] = useState<ConfigField[]>([]);
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState<Record<string, unknown>>({});

  // Load config schema when agent changes
  useEffect(() => {
    let cancelled = false;

    

    const loadOptions = async () => {
      if (!agentSlug) {
        setFields([]);
        setConfig({});
        return;
      }

      setLoading(true);
      try {
        // Load config schema from Backend
        const schemaResponse = await agentApi.getConfigSchema(agentSlug);

        if (cancelled) return;

        const schema = schemaResponse.schema || { fields: [] };
        const baseFields = schema.fields || [];

        // Step 1: Initialize config with ConfigSchema defaults
        const mergedConfig: Record<string, unknown> = {};
        for (const field of baseFields) {
          if (field.default !== undefined) {
            mergedConfig[field.name] = field.default;
          }
        }

        // Step 2: Load user personal config and merge (higher priority)
        try {
          const userConfigResponse = await userAgentConfigApi.get(agentSlug);
          if (!cancelled && userConfigResponse.config?.config_values) {
            const userConfig = userConfigResponse.config.config_values;

            // Merge user config into mergedConfig
            for (const field of baseFields) {
              if (userConfig[field.name] !== undefined) {
                mergedConfig[field.name] = userConfig[field.name];
              }
            }
          }
        } catch (err) {
          // User config not found or error - use ConfigSchema defaults only
        }

        if (!cancelled) {
          // Step 3: Derive `model` field from `models` list if present
          const modelsField = baseFields.find((f) => f.type === "model_list");
          const modelField = modelsField ? deriveModelField(mergedConfig[modelsField.name]) : null;

          // Combine base fields with derived model field
          const allFields = modelField ? [...baseFields, modelField] : baseFields;

          setFields(allFields);
          setConfig(mergedConfig);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to load config schema:", err);
        setFields([]);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadOptions();

    return () => {
      cancelled = true;
    };
  }, [agentSlug]);

  // Update a single config field
  const updateConfig = useCallback(
    (fieldName: string, value: unknown) => {
      setConfig((prev) => {
        const updated = { ...prev, [fieldName]: value };

        // If models list changed, regenerate model field options
        if (fieldName === "models") {
          const modelsField = fields.find((f) => f.type === "model_list");
          if (modelsField) {
            const modelField = deriveModelField(value);
            if (modelField) {
              // Remove old model from config if it's not in the new models list
              const modelList = Array.isArray(value) ? value.filter((m): m is string => typeof m === "string") : [];
              const currentModel = prev["model"] as string;
              if (currentModel && !modelList.includes(currentModel)) {
                updated["model"] = "";
              }
              // Update fields to include new model options
              setFields((currentFields) => {
                const baseFields = currentFields.filter((f) => f.name !== "model");
                return [...baseFields, modelField];
              });
            }
          }
        }

        return updated;
      });
    },
    [fields]
  );

  // Reset config to empty
  const resetConfig = useCallback(() => {
    setConfig({});
    setFields([]);
  }, []);

  return { fields, loading, config, updateConfig, resetConfig };
}
