/**
 * BillingLoadingSkeleton - Loading state for billing settings
 */
export function BillingLoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6 animate-pulse">
        <div className="h-6 bg-muted rounded w-32 mb-4"></div>
        <div className="h-8 bg-muted rounded w-48 mb-2"></div>
        <div className="h-4 bg-muted rounded w-64"></div>
      </div>
    </div>
  );
}
