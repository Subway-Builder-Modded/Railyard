import { create } from "zustand";
import { DEFAULT_SORT_STATE, type SortState } from "@/lib/constants";
import {
  createDefaultSharedAssetFilters,
  type SharedAssetFilterState,
} from "@/stores/asset-filter-state";

export type TypeFilter = SharedAssetFilterState["type"];

export interface SearchFilterState extends SharedAssetFilterState {
  sort: SortState;
  randomSeed: number;
}

type SearchFilterUpdater =
  | SearchFilterState
  | ((prev: SearchFilterState) => SearchFilterState);

interface SearchState {
  filters: SearchFilterState;
  page: number;
  setFilters: (updater: SearchFilterUpdater) => void;
  setPage: (page: number) => void;
}

export function createRandomSeed(): number {
  return Math.floor(Math.random() * 2_147_483_647);
}

const defaultSearchFilters: SearchFilterState = {
  ...createDefaultSharedAssetFilters("map"),
  sort: DEFAULT_SORT_STATE,
  randomSeed: createRandomSeed(),
};

export const useSearchStore = create<SearchState>((set) => ({
  filters: defaultSearchFilters,
  page: 1,
  setFilters: (updater) =>
    set((state) => ({
      filters: typeof updater === "function" ? updater(state.filters) : updater,
    })),
  setPage: (page) => set({ page }),
}));
