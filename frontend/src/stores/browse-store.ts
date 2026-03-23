import { create } from 'zustand';

import type { SearchViewMode } from '@/lib/search-view-mode';
import type {
  AssetQueryFilters,
  AssetQueryFilterStoreState,
  AssetQueryFilterUpdater,
} from '@/stores/asset-query-filter-store';
import {
  cloneFilterState,
  createFilterByAssetType,
  createRandomSeed,
  defaultSearchFilters,
  switchFilter,
  syncFilter,
} from '@/stores/asset-type-filter-state';

export { createRandomSeed };

export type BrowseFilterState = AssetQueryFilters;
export type BrowseFilterUpdater = AssetQueryFilterUpdater;
export type BrowseFilterStoreState = AssetQueryFilterStoreState;

interface BrowseViewModeStoreState {
  viewMode: SearchViewMode;
  viewModeInitialized: boolean;
  setViewMode: (viewMode: SearchViewMode) => void;
  initializeViewMode: (viewMode: SearchViewMode) => void;
}

export const useBrowseStore = create<
  BrowseFilterStoreState & BrowseViewModeStoreState
>((set, get) => ({
  filters: cloneFilterState(defaultSearchFilters),
  page: 1,
  scopedByType: createFilterByAssetType(defaultSearchFilters, 1),
  viewMode: 'full',
  viewModeInitialized: false,
  setFilters: (updater) =>
    set((state) => {
      const nextFilters =
        typeof updater === 'function' ? updater(state.filters) : updater;
      return {
        filters: nextFilters,
        scopedByType: syncFilter(state.scopedByType, nextFilters, state.page),
      };
    }),
  setType: (type) =>
    set((state) =>
      switchFilter(state.filters, state.page, state.scopedByType, type),
    ),
  setPage: (page) =>
    set((state) => ({
      page,
      scopedByType: syncFilter(state.scopedByType, state.filters, page),
    })),
  setViewMode: (viewMode) => set({ viewMode, viewModeInitialized: true }),
  initializeViewMode: (viewMode) => {
    if (get().viewModeInitialized) return;
    set({ viewMode, viewModeInitialized: true });
  },
}));
