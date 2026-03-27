"use client";

import { useState } from "react";
import { CreditCard, AlertTriangle, Plus } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { createSubscription } from "@/lib/api/admin";
import type { SubscriptionPlan } from "@/lib/api/admin";

export function NoSubscriptionPanel({
  plans,
  orgId,
  onCreated,
}: {
  plans: SubscriptionPlan[];
  orgId: number;
  onCreated: () => void;
}) {
  const [selectedPlan, setSelectedPlan] = useState(plans[0]?.name || "based");
  const [months, setMonths] = useState("1");
  const [isCreating, setIsCreating] = useState(false);

  const handleCreate = async () => {
    const m = parseInt(months);
    if (m <= 0 || m > 120) return;
    if (!confirm(`Create an active "${selectedPlan}" subscription for ${m} month(s)?`)) return;
    setIsCreating(true);
    try {
      await createSubscription(orgId, selectedPlan, m);
      toast.success("Subscription created");
      onCreated();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to create subscription");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CreditCard className="h-5 w-5" />
          Subscription
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="flex items-center gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950">
            <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
            <div>
              <p className="text-sm font-medium text-amber-800 dark:text-amber-200">
                No subscription record found
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-400">
                This organization is missing a subscription record. Use the form below to create one.
              </p>
            </div>
          </div>

          <div className="space-y-3 rounded-lg border border-border p-4">
            <h3 className="text-sm font-semibold text-muted-foreground">Create Subscription</h3>
            <div className="flex items-center justify-between">
              <span className="text-sm">Plan</span>
              <Select value={selectedPlan} onValueChange={setSelectedPlan}>
                <SelectTrigger className="h-8 w-40 text-sm">
                  <SelectValue placeholder="Select plan" />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((p) => (
                    <SelectItem key={p.name} value={p.name}>
                      {p.display_name || p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">Duration</span>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  min={1}
                  max={120}
                  value={months}
                  onChange={(e) => setMonths(e.target.value)}
                  className="h-8 w-20 text-sm"
                />
                <span className="text-xs text-muted-foreground">months</span>
              </div>
            </div>
            <Button
              className="w-full"
              disabled={isCreating || !selectedPlan}
              onClick={handleCreate}
            >
              <Plus className="mr-2 h-4 w-4" />
              Create Subscription
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
