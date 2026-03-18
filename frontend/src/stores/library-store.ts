import { create } from 'zustand';

import {
  cloneFilterState,
  createFilterByAssetType,
  defaultLibraryFilters,
  switchFilter,
  syncFilter,
} from '@/stores/asset-type-filter-state';
import { type SearchFilterStoreState } from '@/stores/search-store';

interface LibraryState extends SearchFilterStoreState {
  selectedIds: Set<string>;
  toggleSelected: (id: string) => void;
  selectAll: (ids: string[]) => void;
  removeSelected: (ids: string[]) => void;
  clearSelection: () => void;
  isSelected: (id: string) => boolean;
}

export const useLibraryStore = create<LibraryState>((set, get) => ({
  filters: cloneFilterState(defaultLibraryFilters),
  page: 1,
  scopedByType: createFilterByAssetType(defaultLibraryFilters, 1),
  selectedIds: new Set<string>(),
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
  removeSelected: (ids) =>
    set((state) => {
      if (ids.length === 0) return state;

      const next = new Set(state.selectedIds);
      for (const id of ids) {
        next.delete(id);
      }

      return { selectedIds: next };
    }),
  clearSelection: () => set({ selectedIds: new Set() }),
  isSelected: (id) => get().selectedIds.has(id),
}));
