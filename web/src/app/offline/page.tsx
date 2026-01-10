"use client";

import { WifiOff, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function OfflinePage() {
  const handleRetry = () => {
    window.location.reload();
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="text-center max-w-md">
        <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-muted flex items-center justify-center">
          <WifiOff className="w-10 h-10 text-muted-foreground" />
        </div>

        <h1 className="text-2xl font-bold mb-2">You're Offline</h1>
        <p className="text-muted-foreground mb-6">
          It looks like you've lost your internet connection. Please check your
          connection and try again.
        </p>

        <Button onClick={handleRetry} className="gap-2">
          <RefreshCw className="w-4 h-4" />
          Try Again
        </Button>

        <div className="mt-8 p-4 bg-muted rounded-lg text-sm text-muted-foreground">
          <p className="font-medium mb-2">While offline, you can:</p>
          <ul className="text-left space-y-1">
            <li>• View previously loaded pages</li>
            <li>• Access cached terminal sessions</li>
            <li>• Review offline-available tickets</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
