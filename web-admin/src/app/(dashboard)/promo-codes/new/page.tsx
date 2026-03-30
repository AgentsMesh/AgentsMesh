"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { createPromoCode } from "@/lib/api/admin";
import {
  PromoCodeForm,
  type PromoCodeFormData,
} from "./_components/promo-code-form";

export default function NewPromoCodePage() {
  const router = useRouter();
  const [isCreating, setIsCreating] = useState(false);
  const [formData, setFormData] = useState<PromoCodeFormData>({
    code: "",
    name: "",
    description: "",
    type: "campaign",
    plan_name: "pro",
    duration_months: 1,
    max_uses: "",
    max_uses_per_org: 1,
    expires_at: "",
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const data = {
      code: formData.code.toUpperCase(),
      name: formData.name,
      description: formData.description || undefined,
      type: formData.type,
      plan_name: formData.plan_name,
      duration_months: formData.duration_months,
      max_uses: formData.max_uses ? parseInt(formData.max_uses) : undefined,
      max_uses_per_org: formData.max_uses_per_org,
      expires_at: formData.expires_at
        ? new Date(formData.expires_at).toISOString()
        : undefined,
    };

    setIsCreating(true);
    try {
      await createPromoCode(data);
      toast.success("Promo code created successfully");
      router.push("/promo-codes");
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to create promo code");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/promo-codes">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Create Promo Code</h1>
          <p className="text-sm text-muted-foreground">
            Create a new promotional code for subscriptions
          </p>
        </div>
      </div>

      {/* Form */}
      <PromoCodeForm
        formData={formData}
        isCreating={isCreating}
        onFormChange={setFormData}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
