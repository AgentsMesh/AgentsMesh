"use client";

import type { TokenUsageByModel } from "@/lib/api/token-usage";
import type { TranslationFn } from "../GeneralSettings";
import { formatTokenCount, formatNumber } from "./format";

interface UsageByModelTableProps {
  data: TokenUsageByModel[];
  t: TranslationFn;
}

export function UsageByModelTable({ data, t }: UsageByModelTableProps) {
  const sorted = [...data].sort((a, b) => b.total_tokens - a.total_tokens);

  return (
    <div className="border border-border rounded-lg p-6">
      <h3 className="text-sm font-medium mb-4">{t("settings.usagePage.byModelTitle")}</h3>
      {sorted.length === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-8">
          {t("settings.usagePage.noData")}
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <caption className="sr-only">{t("settings.usagePage.byModelTitle")}</caption>
            <thead>
              <tr className="border-b border-border text-left">
                <th scope="col" className="pb-2 font-medium text-muted-foreground">
                  {t("settings.usagePage.columnModel")}
                </th>
                <th scope="col" className="pb-2 font-medium text-muted-foreground text-right">
                  {t("settings.usagePage.inputTokens")}
                </th>
                <th scope="col" className="pb-2 font-medium text-muted-foreground text-right">
                  {t("settings.usagePage.outputTokens")}
                </th>
                <th scope="col" className="pb-2 font-medium text-muted-foreground text-right">
                  {t("settings.usagePage.totalTokens")}
                </th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((model) => (
                <tr key={model.model} className="border-b border-border/50 last:border-0">
                  <td className="py-2 font-medium">{model.model?.trim() || t("settings.usagePage.unknown")}</td>
                  <td className="py-2 text-right" title={formatNumber(model.input_tokens)}>
                    {formatTokenCount(model.input_tokens)}
                  </td>
                  <td className="py-2 text-right" title={formatNumber(model.output_tokens)}>
                    {formatTokenCount(model.output_tokens)}
                  </td>
                  <td className="py-2 text-right font-medium" title={formatNumber(model.total_tokens)}>
                    {formatTokenCount(model.total_tokens)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
