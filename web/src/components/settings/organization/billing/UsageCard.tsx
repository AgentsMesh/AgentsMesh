import type { BillingOverview } from "@/lib/api";
import type { TranslationFn } from "../GeneralSettings";

interface UsageCardProps {
  usage: BillingOverview["usage"];
  getUsagePercent: (current: number, max: number) => number;
  formatLimit: (value: number) => string;
  t: TranslationFn;
}

/**
 * UsageCard - Displays current resource usage with progress bars
 */
export function UsageCard({
  usage,
  getUsagePercent,
  formatLimit,
  t,
}: UsageCardProps) {
  const usageItems = [
    { label: t("settings.billingPage.podMinutes"), current: Math.round(usage.pod_minutes), max: usage.included_pod_minutes },
    { label: t("settings.billingPage.teamMembers"), current: usage.users, max: usage.max_users },
    { label: "Runners", current: usage.runners, max: usage.max_runners },
    { label: t("settings.billingPage.repositories"), current: usage.repositories, max: usage.max_repositories },
  ];

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.usage")}</h2>
      <div className="space-y-4">
        {usageItems.map((item, index) => (
          <div key={index}>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">{item.label}</span>
              <span className="text-sm font-medium">
                {item.current} / {formatLimit(item.max)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(item.current, item.max) > 90 ? "bg-destructive" : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(item.current, item.max)}%` }}
              ></div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
