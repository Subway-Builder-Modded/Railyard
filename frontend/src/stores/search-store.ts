import { create } from 'zustand';

import type { AssetType } from '@/lib/asset-types';
import {
  DEFAULT_SORT_STATE,
  type PerPage,
  type SortState,
} from '@/lib/constants';
import {
  createTypeScopedByAssetType,
  switchTypeScopedState,
  syncCurrentTypeScopedState,
  type TypeScopedByAssetType,
} from '@/stores/asset-type-filter-state';

export interface SearchFilterState {
  query: string;
  type: AssetType;
  sort: SortState;
  randomSeed: number;
  perPage: PerPage;
  mod: {
    tags: string[];
  };
  map: {
    locations: string[];
    sourceQuality: string[];
    levelOfDetail: string[];
    specialDemand: string[];
  };
}

export type SearchFilterUpdater =
  | SearchFilterState
  | ((prev: SearchFilterState) => SearchFilterState);

export interface SearchFilterStoreState {
  filters: SearchFilterState;
  page: number;
  scopedByType: TypeScopedByAssetType;
  setFilters: (updater: SearchFilterUpdater) => void;
  setType: (type: AssetType) => void;
  setPage: (page: number) => void;
}

export function createRandomSeed(): number {
  return Math.floor(Math.random() * 2_147_483_647);
}

const defaultSearchFilters: SearchFilterState = {
  query: '',
  type: 'map',
  sort: DEFAULT_SORT_STATE,
  randomSeed: createRandomSeed(),
  perPage: 12,
  mod: {
    tags: [],
  },
  map: {
    locations: [],
    sourceQuality: [],
    levelOfDetail: [],
    specialDemand: [],
  },
};

const defaultSearchScopedByType = createTypeScopedByAssetType(
  defaultSearchFilters,
  1,
);

export const useSearchStore = create<SearchFilterStoreState>((set) => ({
  filters: defaultSearchFilters,
  page: 1,
  scopedByType: defaultSearchScopedByType,
  setFilters: (updater) =>
    set((state) => {
      const nextFilters =
        typeof updater === 'function' ? updater(state.filters) : updater;
      return {
        filters: nextFilters,
        scopedByType: syncCurrentTypeScopedState(
          state.scopedByType,
          nextFilters,
          state.page,
        ),
      };
    }),
  setType: (type) =>
    set((state) => switchTypeScopedState(state.filters, state.page, state.scopedByType, type)),
  setPage: (page) =>
    set((state) => ({
      page,
      scopedByType: syncCurrentTypeScopedState(
        state.scopedByType,
        state.filters,
        page,
      ),
    })),
}));
