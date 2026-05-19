import { useState, useEffect } from "react";
import { getServerUrl, getServerUrlSSR } from "@/lib/env";

export function useServerUrl(): string {
  const [serverUrl, setServerUrl] = useState(getServerUrlSSR);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setServerUrl(getServerUrl());
  }, []);

  return serverUrl;
}
