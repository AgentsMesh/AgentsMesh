import { useState, useEffect, useCallback } from "react";
import { agentApi, userAgentConfigApi, ConfigField } from "@/lib/api/client";

export interface ConfigOptionsState {
  fields: ConfigField[];
  loading: boolean;
  config: Record<string, unknown>;
  updateConfig: (fieldName: string, value: unknown) => void;
  resetConfig: () => void;
}

/**
 * Hook to manage agent config options and configuration
 * Loads config schema from Backend when agent type is selected
 *
 * Configuration priority (high to low):
 * 1. User overrides in the form
 * 2. User personal config (from personal settings)
 * 3. Backend ConfigSchema defaults
 */
export function useConfigOptions(
  runnerId: number | null,
  agentSlug: string,
  agentTypeId?: number | null
): ConfigOptionsState {
  const [fields, setFields] = useState<ConfigField[]>([]);
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState<Record<string, unknown>>({});

  // Load config schema when agent type changes
  useEffect(() => {
    let cancelled = false;

    console.log("[useConfigOptions] agentTypeId:", agentTypeId, "agentSlug:", agentSlug);

    const loadOptions = async () => {
      if (!agentTypeId) {
        console.log("[useConfigOptions] Skipping - missing agentTypeId");
        setFields([]);
        setConfig({});
        return;
      }

      setLoading(true);
      try {
        // Load config schema from Backend
        const schemaResponse = await agentApi.getConfigSchema(agentTypeId);

        if (cancelled) return;

        const schema = schemaResponse.schema || { fields: [] };
        setFields(schema.fields || []);

        // Step 1: Initialize config with ConfigSchema defaults
        const mergedConfig: Record<string, unknown> = {};
        for (const field of schema.fields || []) {
          if (field.default !== undefined) {
            mergedConfig[field.name] = field.default;
          }
        }

        // Step 2: Load user personal config and merge (higher priority)
        try {
          const userConfigResponse = await userAgentConfigApi.get(agentTypeId);
          if (!cancelled && userConfigResponse.config?.config_values) {
            const userConfig = userConfigResponse.config.config_values;
            console.log("[useConfigOptions] User personal config:", userConfig);

            // Merge user config into mergedConfig
            for (const field of schema.fields || []) {
              if (userConfig[field.name] !== undefined) {
                console.log("[useConfigOptions] Merging user config:", field.name, "=", userConfig[field.name]);
                mergedConfig[field.name] = userConfig[field.name];
              }
            }
            console.log("[useConfigOptions] Final merged config:", mergedConfig);
          }
        } catch (err) {
          // User config not found or error - use ConfigSchema defaults only
          console.log("[useConfigOptions] No user personal config found, using ConfigSchema defaults", err);
        }

        if (!cancelled) {
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
  }, [agentTypeId, agentSlug]);

  // Update a single config field
  const updateConfig = useCallback(
    (fieldName: string, value: unknown) => {
      setConfig((prev) => ({
        ...prev,
        [fieldName]: value,
      }));
    },
    []
  );

  // Reset config to empty
  const resetConfig = useCallback(() => {
    setConfig({});
    setFields([]);
  }, []);

  return { fields, loading, config, updateConfig, resetConfig };
}
