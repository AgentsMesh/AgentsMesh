"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { billingApi, Invoice } from "@/lib/api/billing";

interface InvoiceHistoryProps {
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function InvoiceHistory({ t }: InvoiceHistoryProps) {
  const [loading, setLoading] = useState(true);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(0);
  const [hasMore, setHasMore] = useState(true);

  const pageSize = 10;

  useEffect(() => {
    loadInvoices(0);
  }, []);

  const loadInvoices = async (pageNum: number) => {
    setLoading(true);
    setError(null);
    try {
      const response = await billingApi.listInvoices(pageSize, pageNum * pageSize);
      if (pageNum === 0) {
        setInvoices(response.invoices);
      } else {
        setInvoices((prev) => [...prev, ...response.invoices]);
      }
      setHasMore(response.invoices.length === pageSize);
      setPage(pageNum);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load invoices");
    } finally {
      setLoading(false);
    }
  };

  const loadMore = () => {
    loadInvoices(page + 1);
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case "paid":
        return "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400";
      case "pending":
        return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400";
      case "failed":
      case "cancelled":
        return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";
      default:
        return "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300";
    }
  };

  const formatCurrency = (amount: number, currency: string) => {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: currency.toUpperCase(),
    }).format(amount);
  };

  if (error && invoices.length === 0) {
    return (
      <div className="border border-border rounded-lg p-6">
        <p className="text-destructive">{error}</p>
        <Button variant="outline" className="mt-4" onClick={() => loadInvoices(0)}>
          {t("billing.invoices.retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">
        {t("billing.invoices.title")}
      </h2>

      {invoices.length === 0 && !loading ? (
        <p className="text-muted-foreground text-center py-8">
          {t("billing.invoices.noInvoices")}
        </p>
      ) : (
        <div className="space-y-4">
          {/* Invoice List */}
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                    {t("billing.invoices.invoiceNo")}
                  </th>
                  <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                    {t("billing.invoices.date")}
                  </th>
                  <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                    {t("billing.invoices.period")}
                  </th>
                  <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                    {t("billing.invoices.amount")}
                  </th>
                  <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                    {t("billing.invoices.status")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {invoices.map((invoice) => (
                  <tr
                    key={invoice.id}
                    className="border-b border-border last:border-0"
                  >
                    <td className="py-3 px-2">
                      <span className="font-mono text-sm">
                        {invoice.invoice_no}
                      </span>
                    </td>
                    <td className="py-3 px-2 text-sm">
                      {new Date(invoice.created_at).toLocaleDateString()}
                    </td>
                    <td className="py-3 px-2 text-sm text-muted-foreground">
                      {new Date(invoice.billing_period_start).toLocaleDateString()}{" "}
                      -{" "}
                      {new Date(invoice.billing_period_end).toLocaleDateString()}
                    </td>
                    <td className="py-3 px-2 text-sm text-right font-medium">
                      {formatCurrency(invoice.total_amount, invoice.currency)}
                    </td>
                    <td className="py-3 px-2 text-right">
                      <span
                        className={`text-xs px-2 py-1 rounded ${getStatusBadgeClass(
                          invoice.status
                        )}`}
                      >
                        {invoice.status.charAt(0).toUpperCase() +
                          invoice.status.slice(1)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Load More */}
          {hasMore && (
            <div className="text-center pt-4">
              <Button
                variant="outline"
                onClick={loadMore}
                loading={loading}
              >
                {loading
                  ? t("billing.invoices.loading")
                  : t("billing.invoices.loadMore")}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
