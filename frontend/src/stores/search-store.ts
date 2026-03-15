import { create } from 'zustand';

import type { AssetType } from '@/lib/asset-types';
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

export const useSearchStore = create<SearchFilterStoreState>((set) => ({
  filters: cloneFilterState(defaultSearchFilters),
  page: 1,
  scopedByType: createFilterByAssetType(defaultSearchFilters, 1),
  setFilters: (updater) =>
    set((state) => {
      const nextFilters =
        typeof updater === 'function' ? updater(state.filters) : updater;
      return {
        filters: nextFilters,
        scopedByType: syncFilter(
          state.scopedByType,
          nextFilters,
          state.page,
        ),
      };
    }),
  setType: (type) =>
    set((state) => switchFilter(state.filters, state.page, state.scopedByType, type)),
  setPage: (page) =>
    set((state) => ({
      page,
      scopedByType: syncFilter(
        state.scopedByType,
        state.filters,
        page,
      ),
    })),
}));
