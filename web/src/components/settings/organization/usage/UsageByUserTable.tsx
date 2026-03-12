"use client";

import type { TokenUsageByUser } from "@/lib/api/token-usage";
import type { TranslationFn } from "../GeneralSettings";
import { formatTokenCount, formatNumber } from "./format";

interface UsageByUserTableProps {
  data: TokenUsageByUser[];
  t: TranslationFn;
}

export function UsageByUserTable({ data, t }: UsageByUserTableProps) {
  const sorted = [...data].sort((a, b) => b.total_tokens - a.total_tokens);

  return (
    <div className="border border-border rounded-lg p-6">
      <h3 className="text-sm font-medium mb-4">{t("settings.usagePage.byUserTitle")}</h3>
      {sorted.length === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-8">
          {t("settings.usagePage.noData")}
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <caption className="sr-only">{t("settings.usagePage.byUserTitle")}</caption>
            <thead>
              <tr className="border-b border-border text-left">
                <th scope="col" className="pb-2 font-medium text-muted-foreground">
                  {t("settings.usagePage.columnUser")}
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
              {sorted.map((user) => (
                <tr key={user.user_id} className="border-b border-border/50 last:border-0">
                  <td className="py-2">
                    <div className="font-medium">{user.username}</div>
                    <div className="text-xs text-muted-foreground">{user.email}</div>
                  </td>
                  <td className="py-2 text-right" title={formatNumber(user.input_tokens)}>
                    {formatTokenCount(user.input_tokens)}
                  </td>
                  <td className="py-2 text-right" title={formatNumber(user.output_tokens)}>
                    {formatTokenCount(user.output_tokens)}
                  </td>
                  <td className="py-2 text-right font-medium" title={formatNumber(user.total_tokens)}>
                    {formatTokenCount(user.total_tokens)}
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
