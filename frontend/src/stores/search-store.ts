import { create } from "zustand";
import { type PerPage, type SortOption } from "@/lib/constants";

export type TypeFilter = "mods" | "maps";

export interface SearchFilterState {
  query: string;
  type: TypeFilter;
  sort: SortOption;
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

type SearchFilterUpdater = SearchFilterState | ((prev: SearchFilterState) => SearchFilterState);

interface SearchState {
  filters: SearchFilterState;
  page: number;
  setFilters: (updater: SearchFilterUpdater) => void;
  setPage: (page: number) => void;
}

const defaultSearchFilters: SearchFilterState = {
  query: "",
  type: "maps",
  sort: "name-asc",
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

export const useSearchStore = create<SearchState>((set) => ({
  filters: defaultSearchFilters,
  page: 1,
  setFilters: (updater) =>
    set((state) => ({
      filters: typeof updater === "function" ? updater(state.filters) : updater,
    })),
  setPage: (page) => set({ page }),
}));
