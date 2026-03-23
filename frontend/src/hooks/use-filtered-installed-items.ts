import { useMemo } from 'react';

import {
  filterAndSortTaggedItems,
  type TaggedItemFilterState,
} from '@/hooks/use-filtered-items';
import { usePaginationSync } from '@/hooks/use-pagination-sync';
import { useLibraryStore } from '@/stores/library-store';
import { useProfileStore } from '@/stores/profile-store';

import type { types } from '../../wailsjs/go/models';
import { type PerPage } from '../lib/constants';

export type InstalledTaggedItem =
  | {
      type: 'mod';
      item: types.ModManifest;
      installedVersion: string;
      isLocal: boolean;
    }
  | {
      type: 'map';
      item: types.MapManifest;
      installedVersion: string;
      isLocal: boolean;
    };

interface UseFilteredInstalledParams {
  items: InstalledTaggedItem[];
  modDownloadTotals: Record<string, number>;
  mapDownloadTotals: Record<string, number>;
}

export function useFilteredInstalledItems({
  items,
  modDownloadTotals,
  mapDownloadTotals,
}: UseFilteredInstalledParams) {
  const defaultPerPage = useProfileStore((s) => s.defaultPerPage)() as PerPage;
  const filters = useLibraryStore((s) => s.filters);
  const setFilters = useLibraryStore((s) => s.setFilters);
  const setType = useLibraryStore((s) => s.setType);
  const page = useLibraryStore((s) => s.page);
  const setPage = useLibraryStore((s) => s.setPage);

  usePaginationSync({ defaultPerPage, filters, setFilters, setPage });

  const filtered = useMemo(() => {
    return filterAndSortTaggedItems(
      items,
      filters as TaggedItemFilterState,
      modDownloadTotals,
      mapDownloadTotals,
    );
  }, [items, filters, mapDownloadTotals, modDownloadTotals]);

  const totalResults = filtered.length;
  const totalPages = Math.max(1, Math.ceil(totalResults / filters.perPage));

  const paginatedItems = useMemo(() => {
    const start = (page - 1) * filters.perPage;
    return filtered.slice(start, start + filters.perPage);
  }, [filtered, page, filters.perPage]);

  return {
    items: paginatedItems,
    allFilteredItems: filtered,
    page,
    totalPages,
    totalResults,
    filters,
    setFilters,
    setType,
    setPage,
  };
}
