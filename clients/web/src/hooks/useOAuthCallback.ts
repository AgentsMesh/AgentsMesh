"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  consumeOAuthCallbackParams,
  resolvePostLoginUrlLight,
} from "@/lib/light-auth";

export type OAuthCallbackStatus = "loading" | "success" | "error";

// processedRef gates strict-mode double-mount. The redirect timer survives unmount
// intentionally — clearing it would let strict-mode kill the redirect before the
// second mount short-circuits, stranding the user on the loading screen.
export function useOAuthCallback(searchParams: {
  get(key: string): string | null;
}): { status: OAuthCallbackStatus; errorReason: string } {
  const router = useRouter();
  const [status, setStatus] = useState<OAuthCallbackStatus>("loading");
  const [errorReason, setErrorReason] = useState<string>("");
  const processedRef = useRef(false);

  useEffect(() => {
    if (processedRef.current) return;
    processedRef.current = true;

    const hadSensitive =
      searchParams.get("token") ||
      searchParams.get("refresh_token") ||
      searchParams.get("error");
    if (hadSensitive && typeof window !== "undefined") {
      window.history.replaceState({}, "", window.location.pathname);
    }

    const redirectParam = searchParams.get("redirect");

    (async () => {
      const result = consumeOAuthCallbackParams(searchParams);
      if (result.status === "error") {
        setStatus("error");
        setErrorReason(result.reason);
        return;
      }
      try {
        const url = await resolvePostLoginUrlLight({ redirectParam });
        setStatus("success");
        setTimeout(() => router.push(url), 1500);
      } catch {
        setStatus("error");
        setErrorReason("authentication_failed");
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- processedRef gates single execution
  }, []);

  return { status, errorReason };
}
