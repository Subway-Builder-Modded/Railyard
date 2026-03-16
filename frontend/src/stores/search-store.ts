import { create } from 'zustand';

import type { AssetType } from '@/lib/asset-types';
import type { SearchViewMode } from '@/lib/search-view-mode';
import {
  type AssetQueryFilterState,
  cloneFilterState,
  createFilterByAssetType,
  createRandomSeed,
  defaultSearchFilters,
  type FilterByAssetType,
  switchFilter,
  syncFilter,
} from '@/stores/asset-type-filter-state';

export { createRandomSeed };

export type SearchFilterState = AssetQueryFilterState;

export type SearchFilterUpdater =
  | SearchFilterState
  | ((prev: SearchFilterState) => SearchFilterState);

export interface SearchFilterStoreState {
  filters: SearchFilterState;
  page: number;
  scopedByType: FilterByAssetType;
  setFilters: (updater: SearchFilterUpdater) => void;
  setType: (type: AssetType) => void;
  setPage: (page: number) => void;
}

interface SearchViewModeStoreState {
  viewMode: SearchViewMode;
  viewModeInitialized: boolean;
  setViewMode: (viewMode: SearchViewMode) => void;
  initializeViewMode: (viewMode: SearchViewMode) => void;
}

export const useSearchStore = create<
  SearchFilterStoreState & SearchViewModeStoreState
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
