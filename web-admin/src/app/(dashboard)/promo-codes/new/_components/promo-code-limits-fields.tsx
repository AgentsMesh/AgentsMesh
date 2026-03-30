import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { PromoCodeFormData } from "./promo-code-form";

interface PromoCodeLimitsFieldsProps {
  formData: PromoCodeFormData;
  onFormChange: (data: PromoCodeFormData) => void;
}

export function PromoCodeLimitsFields({
  formData,
  onFormChange,
}: PromoCodeLimitsFieldsProps) {
  return (
    <>
      {/* Duration */}
      <div className="space-y-2">
        <Label htmlFor="duration_months">Duration (months) *</Label>
        <Input
          id="duration_months"
          type="number"
          min={1}
          max={24}
          value={formData.duration_months}
          onChange={(e) =>
            onFormChange({
              ...formData,
              duration_months: parseInt(e.target.value) || 1,
            })
          }
          required
        />
        <p className="text-xs text-muted-foreground">
          Number of months the subscription will be extended
        </p>
      </div>

      {/* Usage Limits */}
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="max_uses">Max Total Uses</Label>
          <Input
            id="max_uses"
            type="number"
            min={1}
            placeholder="Unlimited"
            value={formData.max_uses}
            onChange={(e) =>
              onFormChange({ ...formData, max_uses: e.target.value })
            }
          />
          <p className="text-xs text-muted-foreground">
            Leave empty for unlimited uses
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="max_uses_per_org">Max Uses per Organization</Label>
          <Input
            id="max_uses_per_org"
            type="number"
            min={1}
            value={formData.max_uses_per_org}
            onChange={(e) =>
              onFormChange({
                ...formData,
                max_uses_per_org: parseInt(e.target.value) || 1,
              })
            }
          />
        </div>
      </div>

      {/* Expiration */}
      <div className="space-y-2">
        <Label htmlFor="expires_at">Expiration Date</Label>
        <Input
          id="expires_at"
          type="datetime-local"
          value={formData.expires_at}
          onChange={(e) =>
            onFormChange({ ...formData, expires_at: e.target.value })
          }
        />
        <p className="text-xs text-muted-foreground">
          Leave empty for no expiration
        </p>
      </div>
    </>
  );
}
