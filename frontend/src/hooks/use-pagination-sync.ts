import { useEffect, useRef } from 'react';

import type { AssetQueryFilters, AssetQueryFilterUpdater } from '@/stores/asset-query-filter-store';
import type { PerPage } from '@/lib/constants';

interface UsePaginationSyncParams {
  defaultPerPage: PerPage;
  filters: AssetQueryFilters;
  setFilters: (updater: AssetQueryFilterUpdater) => void;
  setPage: (page: number) => void;
}

/**
 * Syncs the per-page setting from the user profile and resets to page 1 on
 * filter changes (but not when the type tab switches, since the store handles
 * that separately with scoped filter state).
 */
export function usePaginationSync({
  defaultPerPage,
  filters,
  setFilters,
  setPage,
}: UsePaginationSyncParams): void {
  useEffect(() => {
    setFilters((prev) =>
      prev.perPage === defaultPerPage ? prev : { ...prev, perPage: defaultPerPage },
    );
  }, [defaultPerPage, setFilters]);

  const didMount = useRef(false);
  const previousTypeRef = useRef(filters.type);
  useEffect(() => {
    if (!didMount.current) {
      didMount.current = true;
      previousTypeRef.current = filters.type;
      return;
    }
    if (previousTypeRef.current !== filters.type) {
      previousTypeRef.current = filters.type;
      return;
    }
    setPage(1);
  }, [filters, setPage]);
}
