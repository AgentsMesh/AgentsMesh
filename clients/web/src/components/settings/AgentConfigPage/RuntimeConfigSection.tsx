"use client";

import { Button } from "@/components/ui/button";
import { ConfigForm } from "@/components/ide/ConfigForm";
import { Settings2 } from "lucide-react";
import type { RuntimeConfigSectionProps } from "./types";

export function RuntimeConfigSection({
  configFields,
  configValues,
  agentSlug,
  saving,
  onChange,
  onSave,
  t,
}: RuntimeConfigSectionProps) {
  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center gap-2 mb-4">
        <Settings2 className="w-5 h-5 text-muted-foreground" />
        <h3 className="text-lg font-semibold">{t("settings.agentConfig.runtime.title")}</h3>
      </div>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.agentConfig.runtime.description")}
      </p>

      {configFields.length > 0 ? (
        <>
          <ConfigForm
            fields={configFields}
            values={configValues}
            onChange={onChange}
            agentSlug={agentSlug}
          />
          <div className="mt-4">
            <Button onClick={onSave} disabled={saving}>
              {saving ? t("common.saving") : t("common.saveChanges")}
            </Button>
          </div>
        </>
      ) : (
        <div className="text-center py-8 text-muted-foreground">
          {t("settings.agentConfig.noConfigFields")}
        </div>
      )}
    </div>
  );
}

export default RuntimeConfigSection;
