import { useState, useEffect, useCallback, useRef } from "react";

export interface AsyncDataState<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
}

export interface UseAsyncDataResult<T> extends AsyncDataState<T> {
  refetch: () => Promise<void>;
  setData: (data: T | null | ((prev: T | null) => T | null)) => void;
  clearError: () => void;
  isLoaded: boolean;
}

export interface UseAsyncDataOptions {
  enabled?: boolean;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
  keepPreviousData?: boolean;
}

export function useAsyncData<T>(
  fetcher: () => Promise<T>,
  deps: React.DependencyList = [],
  options: UseAsyncDataOptions = {}
): UseAsyncDataResult<T> {
  const {
    enabled = true,
    onSuccess,
    onError,
    keepPreviousData = false,
  } = options;

  const [state, setState] = useState<AsyncDataState<T>>({
    data: null,
    loading: enabled,
    error: null,
  });
  const [isLoaded, setIsLoaded] = useState(false);

  const isMountedRef = useRef(true);
  const fetchIdRef = useRef(0);

  const fetchData = useCallback(async () => {
    if (!enabled) return;

    const fetchId = ++fetchIdRef.current;

    setState((prev) => ({
      ...prev,
      loading: true,
      error: null,
      data: keepPreviousData ? prev.data : null,
    }));

    try {
      const result = await fetcher();

      if (isMountedRef.current && fetchId === fetchIdRef.current) {
        setState({
          data: result,
          loading: false,
          error: null,
        });
        setIsLoaded(true);
        onSuccess?.();
      }
    } catch (err) {
      if (isMountedRef.current && fetchId === fetchIdRef.current) {
        const error = err instanceof Error ? err : new Error(String(err));
        setState((prev) => ({
          data: keepPreviousData ? prev.data : null,
          loading: false,
          error,
        }));
        onError?.(error);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, keepPreviousData, ...deps]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const setData = useCallback(
    (dataOrUpdater: T | null | ((prev: T | null) => T | null)) => {
      setState((prev) => ({
        ...prev,
        data:
          typeof dataOrUpdater === "function"
            ? (dataOrUpdater as (prev: T | null) => T | null)(prev.data)
            : dataOrUpdater,
      }));
    },
    []
  );

  const clearError = useCallback(() => {
    setState((prev) => ({ ...prev, error: null }));
  }, []);

  return {
    ...state,
    refetch: fetchData,
    setData,
    clearError,
    isLoaded,
  };
}

export function useAsyncDataAll<T extends Record<string, () => Promise<unknown>>>(
  fetchers: T,
  deps: React.DependencyList = [],
  options: UseAsyncDataOptions = {}
): UseAsyncDataResult<{ [K in keyof T]: Awaited<ReturnType<T[K]>> }> {
  type ResultType = { [K in keyof T]: Awaited<ReturnType<T[K]>> };

  const combinedFetcher = useCallback(async () => {
    const keys = Object.keys(fetchers) as (keyof T)[];
    const promises = keys.map((key) => fetchers[key]());
    const results = await Promise.all(promises);

    return keys.reduce((acc, key, index) => {
      acc[key] = results[index] as Awaited<ReturnType<T[typeof key]>>;
      return acc;
    }, {} as ResultType);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return useAsyncData<ResultType>(combinedFetcher, deps, options);
}
