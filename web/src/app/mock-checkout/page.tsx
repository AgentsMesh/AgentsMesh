"use client";

import { useEffect, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";

interface MockSession {
  session_id: string;
  status: string;
  order_type: string;
  amount: number;
  currency: string;
  created_at: string;
  expires_at: string;
}

export default function MockCheckoutPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [session, setSession] = useState<MockSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [autoComplete, setAutoComplete] = useState(false);

  const sessionId = searchParams.get("session_id");
  const orderNo = searchParams.get("order_no");
  const successUrl = searchParams.get("success_url");
  const cancelUrl = searchParams.get("cancel_url");

  // Fetch session info
  useEffect(() => {
    if (!sessionId) {
      setError("Missing session_id parameter");
      setLoading(false);
      return;
    }

    const fetchSession = async () => {
      try {
        const response = await fetch(`/api/v1/webhooks/mock/session/${sessionId}`);
        if (!response.ok) {
          const data = await response.json();
          throw new Error(data.error || "Failed to fetch session");
        }
        const data = await response.json();
        setSession(data);

        // Check URL params for auto-complete
        if (searchParams.get("auto") === "true") {
          setAutoComplete(true);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to fetch session");
      } finally {
        setLoading(false);
      }
    };

    fetchSession();
  }, [sessionId, searchParams]);

  // Auto-complete if requested
  useEffect(() => {
    if (autoComplete && session && session.status === "pending") {
      handleComplete();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoComplete, session]);

  const handleComplete = async () => {
    if (!sessionId) return;

    setProcessing(true);
    setError(null);

    try {
      const response = await fetch("/api/v1/webhooks/mock/complete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          session_id: sessionId,
          order_no: orderNo,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Payment failed");
      }

      // Redirect to success URL
      if (successUrl) {
        // Add payment=success query param
        const url = new URL(successUrl, window.location.origin);
        url.searchParams.set("payment", "success");
        router.push(url.toString());
      } else {
        // Just show success message
        setSession((prev) => (prev ? { ...prev, status: "succeeded" } : null));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Payment processing failed");
    } finally {
      setProcessing(false);
    }
  };

  const handleCancel = () => {
    if (cancelUrl) {
      const url = new URL(cancelUrl, window.location.origin);
      url.searchParams.set("payment", "cancelled");
      router.push(url.toString());
    } else {
      router.back();
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto"></div>
          <p className="mt-4 text-muted-foreground">Loading checkout...</p>
        </div>
      </div>
    );
  }

  if (error && !session) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="bg-card p-8 rounded-lg shadow-md max-w-md w-full border border-border">
          <div className="text-center">
            <div className="text-destructive text-5xl mb-4">⚠️</div>
            <h1 className="text-xl font-semibold text-foreground mb-2">Checkout Error</h1>
            <p className="text-muted-foreground mb-6">{error}</p>
            <Button variant="outline" onClick={() => router.back()}>
              Go Back
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (session?.status === "succeeded") {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="bg-card p-8 rounded-lg shadow-md max-w-md w-full border border-border">
          <div className="text-center">
            <div className="text-green-500 dark:text-green-400 text-5xl mb-4">✓</div>
            <h1 className="text-xl font-semibold text-foreground mb-2">Payment Successful!</h1>
            <p className="text-muted-foreground mb-6">Your payment has been processed successfully.</p>
            {successUrl && (
              <Button onClick={() => router.push(successUrl)}>
                Continue
              </Button>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="bg-card p-8 rounded-lg shadow-md max-w-md w-full border border-border">
        {/* Header */}
        <div className="text-center mb-6">
          <div className="bg-yellow-500/20 text-yellow-600 dark:text-yellow-400 text-xs font-medium px-3 py-1 rounded-full inline-block mb-4">
            🧪 TEST MODE
          </div>
          <h1 className="text-2xl font-bold text-foreground">Mock Checkout</h1>
          <p className="text-sm text-muted-foreground mt-1">
            This is a simulated payment page for testing
          </p>
        </div>

        {/* Order Details */}
        {session && (
          <div className="border border-border rounded-lg p-4 mb-6">
            <h2 className="font-semibold text-foreground mb-3">Order Details</h2>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Type:</span>
                <span className="font-medium">{session.order_type}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Amount:</span>
                <span className="font-medium">
                  {session.currency.toUpperCase()} {session.amount.toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Session ID:</span>
                <span className="font-mono text-xs">{session.session_id.slice(0, 20)}...</span>
              </div>
              {orderNo && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Order No:</span>
                  <span className="font-mono text-xs">{orderNo}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="bg-destructive/10 text-destructive p-3 rounded-lg mb-4 text-sm">
            {error}
          </div>
        )}

        {/* Simulated Card Input */}
        <div className="border border-border rounded-lg p-4 mb-6 bg-muted">
          <p className="text-sm text-muted-foreground mb-2">Test Card Number</p>
          <div className="font-mono text-lg text-foreground">4242 4242 4242 4242</div>
          <p className="text-xs text-muted-foreground/70 mt-2">
            Use any future date for expiry and any 3 digits for CVC
          </p>
        </div>

        {/* Action Buttons */}
        <div className="space-y-3">
          <Button
            className="w-full"
            size="lg"
            onClick={handleComplete}
            disabled={processing || session?.status !== "pending"}
          >
            {processing ? "Processing..." : "Complete Payment"}
          </Button>
          <Button
            variant="outline"
            className="w-full"
            onClick={handleCancel}
            disabled={processing}
          >
            Cancel
          </Button>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-muted-foreground mt-6">
          This is a mock checkout page. No real payment will be processed.
        </p>
      </div>
    </div>
  );
}
