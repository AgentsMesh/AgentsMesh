"use client";

import { useEffect, useCallback, useRef, useState } from "react";

declare global {
  interface Window {
    createLemonSqueezy?: () => void;
    LemonSqueezy?: {
      Setup: (config: { eventHandler: (event: LemonSqueezyEvent) => void }) => void;
      Url: {
        Open: (url: string) => void;
        Close: () => void;
      };
      Refresh: () => void;
    };
  }
}

export interface LemonSqueezyEvent {
  event:
    | "Checkout.Success"
    | "Checkout.ViewCart"
    | "Checkout.PaymentMethodUpdate"
    | "Checkout.Close"
    | "GA.ViewItem"
    | "PaymentMethodUpdate.Mounted"
    | "PaymentMethodUpdate.Closed"
    | "PaymentMethodUpdate.Updated";
  data?: {
    order?: {
      id: string;
      order_number: string;
      status: string;
    };
    checkout?: {
      id: string;
    };
  };
}

export interface UseLemonSqueezyOptions {
  onCheckoutSuccess?: (event: LemonSqueezyEvent) => void;
  onCheckoutClose?: (event: LemonSqueezyEvent) => void;
}

const LEMON_JS_URL = "https://app.lemonsqueezy.com/js/lemon.js";

function checkScriptLoaded(): boolean {
  if (typeof window === "undefined") return false;
  return !!(window.LemonSqueezy || document.querySelector(`script[src="${LEMON_JS_URL}"]`));
}

export function useLemonSqueezy(options: UseLemonSqueezyOptions = {}) {
  const { onCheckoutSuccess, onCheckoutClose } = options;
  const [isLoaded, setIsLoaded] = useState(checkScriptLoaded);
  const callbacksRef = useRef({ onCheckoutSuccess, onCheckoutClose });

  useEffect(() => {
    callbacksRef.current = { onCheckoutSuccess, onCheckoutClose };
  }, [onCheckoutSuccess, onCheckoutClose]);

  useEffect(() => {
    if (isLoaded) {
      return;
    }

    if (window.LemonSqueezy || document.querySelector(`script[src="${LEMON_JS_URL}"]`)) {
       
      setIsLoaded(true);
      return;
    }

    const script = document.createElement("script");
    script.src = LEMON_JS_URL;
    script.defer = true;
    script.onload = () => {
      if (window.createLemonSqueezy) {
        window.createLemonSqueezy();
      }
      setIsLoaded(true);
    };
    script.onerror = () => {
      console.error("Failed to load LemonSqueezy script");
    };

    document.head.appendChild(script);

    return () => {
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- isLoaded is intentionally not in deps
  }, []);

  useEffect(() => {
    if (!window.LemonSqueezy) return;

    window.LemonSqueezy.Setup({
      eventHandler: (event: LemonSqueezyEvent) => {
        switch (event.event) {
          case "Checkout.Success":
            callbacksRef.current.onCheckoutSuccess?.(event);
            break;
          case "Checkout.Close":
            callbacksRef.current.onCheckoutClose?.(event);
            break;
        }
      },
    });
  }, []);

  const openCheckout = useCallback((url: string) => {
    if (!window.LemonSqueezy) {
      console.warn("LemonSqueezy not loaded, falling back to redirect");
      window.location.href = url;
      return;
    }

    window.LemonSqueezy.Setup({
      eventHandler: (event: LemonSqueezyEvent) => {
        switch (event.event) {
          case "Checkout.Success":
            callbacksRef.current.onCheckoutSuccess?.(event);
            break;
          case "Checkout.Close":
            callbacksRef.current.onCheckoutClose?.(event);
            break;
        }
      },
    });

    window.LemonSqueezy.Url.Open(url);
  }, []);

  const closeCheckout = useCallback(() => {
    if (window.LemonSqueezy) {
      window.LemonSqueezy.Url.Close();
    }
  }, []);

  const refreshCheckout = useCallback(() => {
    if (window.LemonSqueezy) {
      window.LemonSqueezy.Refresh();
    }
  }, []);

  return {
    openCheckout,
    closeCheckout,
    refreshCheckout,
    isLoaded,
  };
}
