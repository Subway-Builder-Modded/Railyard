import { create } from "zustand";
import { type PerPage } from "@/lib/constants";

export type LibraryTypeFilter = "mods" | "maps";
export type LibrarySortOption =
  | "name-asc"
  | "name-desc"
  | "author-asc"
  | "country-asc"
  | "country-desc";

export interface LibraryFilterState {
  query: string;
  type: LibraryTypeFilter;
  sort: LibrarySortOption;
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

type LibraryFilterUpdater =
  | LibraryFilterState
  | ((prev: LibraryFilterState) => LibraryFilterState);

interface LibraryState {
  filters: LibraryFilterState;
  page: number;
  selectedIds: Set<string>;
  setFilters: (updater: LibraryFilterUpdater) => void;
  setPage: (page: number) => void;
  toggleSelected: (id: string) => void;
  selectAll: (ids: string[]) => void;
  clearSelection: () => void;
  isSelected: (id: string) => boolean;
}

const defaultLibraryFilters: LibraryFilterState = {
  query: "",
  type: "mods",
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

export const useLibraryStore = create<LibraryState>((set, get) => ({
  filters: defaultLibraryFilters,
  page: 1,
  selectedIds: new Set<string>(),
  setFilters: (updater) =>
    set((state) => ({
      filters:
        typeof updater === "function" ? updater(state.filters) : updater,
    })),
  setPage: (page) => set({ page }),
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
