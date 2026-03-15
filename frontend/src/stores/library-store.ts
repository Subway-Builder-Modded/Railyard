import { create } from 'zustand';

import {
  DEFAULT_SORT_STATE,
} from '@/lib/constants';
import {
  createTypeScopedByAssetType,
  switchTypeScopedState,
  syncCurrentTypeScopedState,
} from '@/stores/asset-type-filter-state';
import {
  createRandomSeed,
  type SearchFilterState,
  type SearchFilterStoreState,
} from '@/stores/search-store';

interface LibraryState extends SearchFilterStoreState {
  selectedIds: Set<string>;
  toggleSelected: (id: string) => void;
  selectAll: (ids: string[]) => void;
  clearSelection: () => void;
  isSelected: (id: string) => boolean;
}

const defaultLibraryFilters: SearchFilterState = {
  query: '',
  type: 'mod',
  perPage: 12,
  sort: {
    ...DEFAULT_SORT_STATE,
    field: 'name',
    direction: 'asc',
  },
  randomSeed: createRandomSeed(),
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

const defaultLibraryScopedByType = createTypeScopedByAssetType(
  defaultLibraryFilters,
  1,
);

export const useLibraryStore = create<LibraryState>((set, get) => ({
  filters: defaultLibraryFilters,
  page: 1,
  scopedByType: defaultLibraryScopedByType,
  selectedIds: new Set<string>(),
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
    set((state) =>
      switchTypeScopedState(state.filters, state.page, state.scopedByType, type),
    ),
  setPage: (page) =>
    set((state) => ({
      page,
      scopedByType: syncCurrentTypeScopedState(
        state.scopedByType,
        state.filters,
        page,
      ),
    })),
  toggleSelected: (id) =>
    set((state) => {
      const next = new Set(state.selectedIds);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return { selectedIds: next };
    }),
  selectAll: (ids) => set({ selectedIds: new Set(ids) }),
  clearSelection: () => set({ selectedIds: new Set() }),
  isSelected: (id) => get().selectedIds.has(id),
}));
