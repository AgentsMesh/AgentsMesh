"use client";

import { useEffect, useRef } from "react";

interface UseColumnInfiniteScrollOptions {
  hasMore: boolean;
  loading: boolean;
  onLoadMore: () => void;
  root?: Element | null;
}

export function useColumnInfiniteScroll({
  hasMore,
  loading,
  onLoadMore,
  root,
}: UseColumnInfiniteScrollOptions) {
  const sentinelRef = useRef<HTMLDivElement>(null);
  const onLoadMoreRef = useRef(onLoadMore);

  useEffect(() => {
    onLoadMoreRef.current = onLoadMore;
  }, [onLoadMore]);

  const shouldLoad = hasMore && !loading;

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !shouldLoad) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          onLoadMoreRef.current();
        }
      },
      { root: root ?? null, rootMargin: "100px" }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [shouldLoad, root]);

  return sentinelRef;
}
