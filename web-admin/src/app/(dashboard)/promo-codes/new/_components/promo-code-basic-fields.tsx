import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { PromoCodeType } from "@/lib/api/admin";
import type { PromoCodeFormData } from "./promo-code-form";

interface PromoCodeBasicFieldsProps {
  formData: PromoCodeFormData;
  onFormChange: (data: PromoCodeFormData) => void;
}

export function PromoCodeBasicFields({
  formData,
  onFormChange,
}: PromoCodeBasicFieldsProps) {
  return (
    <>
      {/* Code */}
      <div className="space-y-2">
        <Label htmlFor="code">Code *</Label>
        <Input
          id="code"
          placeholder="e.g., SUMMER2024"
          value={formData.code}
          onChange={(e) =>
            onFormChange({ ...formData, code: e.target.value.toUpperCase() })
          }
          required
          minLength={4}
          maxLength={50}
          className="font-mono uppercase"
        />
        <p className="text-xs text-muted-foreground">
          4-50 characters, will be converted to uppercase
        </p>
      </div>

      {/* Name */}
      <div className="space-y-2">
        <Label htmlFor="name">Name *</Label>
        <Input
          id="name"
          placeholder="e.g., Summer Sale 2024"
          value={formData.name}
          onChange={(e) => onFormChange({ ...formData, name: e.target.value })}
          required
          maxLength={100}
        />
      </div>

      {/* Description */}
      <div className="space-y-2">
        <Label htmlFor="description">Description</Label>
        <Textarea
          id="description"
          placeholder="Optional description..."
          value={formData.description}
          onChange={(e) =>
            onFormChange({ ...formData, description: e.target.value })
          }
          rows={3}
        />
      </div>

      {/* Type & Plan */}
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label>Type *</Label>
          <Select
            value={formData.type}
            onValueChange={(value) =>
              onFormChange({ ...formData, type: value as PromoCodeType })
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="media">Media</SelectItem>
              <SelectItem value="partner">Partner</SelectItem>
              <SelectItem value="campaign">Campaign</SelectItem>
              <SelectItem value="internal">Internal</SelectItem>
              <SelectItem value="referral">Referral</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>Plan *</Label>
          <Select
            value={formData.plan_name}
            onValueChange={(value) =>
              onFormChange({ ...formData, plan_name: value })
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pro">Pro</SelectItem>
              <SelectItem value="enterprise">Enterprise</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </>
  );
}
