import { PromoCodeInput } from "@/components/promo-code/PromoCodeInput";
import type { TranslationFn } from "../GeneralSettings";

interface PromoCodeCardProps {
  onRedeemSuccess: () => void;
  t: TranslationFn;
}

/**
 * PromoCodeCard - Card for entering and redeeming promo codes
 */
export function PromoCodeCard({
  onRedeemSuccess,
  t,
}: PromoCodeCardProps) {
  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-2">{t("settings.billingPage.promoCode.title")}</h2>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.billingPage.promoCode.description")}
      </p>
      <PromoCodeInput
        onRedeemSuccess={() => {
          onRedeemSuccess();
        }}
        t={(key: string) => t(`settings.billingPage.promoCode.${key}`)}
      />
    </div>
  );
}
