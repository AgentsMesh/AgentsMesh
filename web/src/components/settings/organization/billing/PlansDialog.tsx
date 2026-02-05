import { Button } from "@/components/ui/button";
import type { SubscriptionPlan } from "@/lib/api";
import type { TranslationFn } from "../GeneralSettings";

interface PlansDialogProps {
  plans: SubscriptionPlan[];
  currentPlan: string | null;
  onSelect: (planName: string) => void;
  onClose: () => void;
  loading: boolean;
  t: TranslationFn;
}

/**
 * PlansDialog - Modal dialog for selecting subscription plans
 */
export function PlansDialog({
  plans,
  currentPlan,
  onSelect,
  onClose,
  loading,
  t,
}: PlansDialogProps) {
  const formatLimit = (value: number): string => {
    return value === -1 ? t("settings.billingPage.unlimited") : String(value);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-4xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold">{t("settings.billingPage.plansDialog.title")}</h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        {plans.length === 0 ? (
          <p className="text-center text-muted-foreground py-8">{t("settings.billingPage.plansDialog.noPlans")}</p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {plans.map((plan) => {
              const isCurrent = plan.name === currentPlan;
              return (
                <div
                  key={plan.id}
                  className={`border rounded-lg p-6 ${
                    isCurrent ? "border-primary bg-primary/5" : "border-border"
                  }`}
                >
                  <div className="mb-4">
                    <h4 className="text-xl font-bold">{plan.display_name}</h4>
                    {plan.price_per_seat_monthly > 0 ? (
                      <p className="text-2xl font-bold mt-2">
                        ${plan.price_per_seat_monthly}
                        <span className="text-sm font-normal text-muted-foreground">/seat/month</span>
                      </p>
                    ) : (
                      <p className="text-2xl font-bold mt-2">{t("settings.billingPage.plansDialog.free")}</p>
                    )}
                  </div>

                  <ul className="space-y-2 mb-6 text-sm">
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.included_pod_minutes)} {t("settings.billingPage.plansDialog.podMinutes")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.max_users)} {t("settings.billingPage.plansDialog.teamMembers")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.max_runners)} {t("settings.billingPage.plansDialog.runners")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.max_repositories)} {t("settings.billingPage.plansDialog.repositories")}
                    </li>
                  </ul>

                  <Button
                    className="w-full"
                    variant={isCurrent ? "outline" : "default"}
                    disabled={isCurrent || loading}
                    onClick={() => onSelect(plan.name)}
                  >
                    {loading ? t("settings.billingPage.plansDialog.processing") : isCurrent ? t("settings.billingPage.plansDialog.currentPlan") : t("settings.billingPage.plansDialog.select")}
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
