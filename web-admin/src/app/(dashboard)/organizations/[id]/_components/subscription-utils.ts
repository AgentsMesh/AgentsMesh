import type { SubscriptionPlan } from "@/lib/api/admin";

export const QUOTA_RESOURCES = ["users", "runners", "concurrent_pods", "repositories", "pod_minutes"];

export function statusVariant(status: string) {
  switch (status) {
    case "active":
      return "success" as const;
    case "frozen":
      return "destructive" as const;
    case "trialing":
      return "default" as const;
    case "canceled":
    case "expired":
      return "secondary" as const;
    default:
      return "outline" as const;
  }
}

export function getPlanLimit(plan: SubscriptionPlan, resource: string): number {
  switch (resource) {
    case "users": return plan.max_users;
    case "runners": return plan.max_runners;
    case "concurrent_pods": return plan.max_concurrent_pods;
    case "repositories": return plan.max_repositories;
    case "pod_minutes": return plan.included_pod_minutes;
    default: return 0;
  }
}
