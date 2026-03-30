"use client";

import Link from "next/link";
import { Tag } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { PromoCodeType } from "@/lib/api/admin";
import { PromoCodeBasicFields } from "./promo-code-basic-fields";
import { PromoCodeLimitsFields } from "./promo-code-limits-fields";

export interface PromoCodeFormData {
  code: string;
  name: string;
  description: string;
  type: PromoCodeType;
  plan_name: string;
  duration_months: number;
  max_uses: string;
  max_uses_per_org: number;
  expires_at: string;
}

interface PromoCodeFormProps {
  formData: PromoCodeFormData;
  isCreating: boolean;
  onFormChange: (data: PromoCodeFormData) => void;
  onSubmit: (e: React.FormEvent) => void;
}

export function PromoCodeForm({
  formData,
  isCreating,
  onFormChange,
  onSubmit,
}: PromoCodeFormProps) {
  return (
    <form onSubmit={onSubmit}>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Tag className="h-5 w-5" />
            Promo Code Details
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <PromoCodeBasicFields formData={formData} onFormChange={onFormChange} />
          <PromoCodeLimitsFields formData={formData} onFormChange={onFormChange} />

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-4">
            <Link href="/promo-codes">
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </Link>
            <Button type="submit" disabled={isCreating}>
              {isCreating ? "Creating..." : "Create Promo Code"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </form>
  );
}
