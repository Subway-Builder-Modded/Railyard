import { useMemo, useEffect, useRef } from "react";
import Fuse from "fuse.js";
import { types } from "../../wailsjs/go/models";
import { type PerPage } from "../lib/constants";
import { FUSE_SEARCH_OPTIONS } from "@/lib/search";
import { useProfileStore } from "@/stores/profile-store";
import { useLibraryStore, type LibrarySortOption } from "@/stores/library-store";

export type InstalledTaggedItem =
  | {
      type: "mod";
      item: types.ModManifest;
      installedVersion: string;
      inRegistry: boolean;
    }
  | {
      type: "map";
      item: types.MapManifest;
      installedVersion: string;
      inRegistry: boolean;
    };

interface UseFilteredInstalledParams {
  items: InstalledTaggedItem[];
}

type SearchableItem = {
  entry: InstalledTaggedItem;
  searchText: string;
};

// Check if a single-valued filter matches (e.g., location, source_quality).
// Empty filter always matches; otherwise value must be in the selected set.
function matchesSingleValueFilter(value: string | undefined, selected: string[]): boolean {
  if (selected.length === 0) return true;
  if (!value) return false;
  return selected.includes(value);
}

// Check if a multi-valued filter matches (e.g., tags, special_demand).
// Empty filter always matches; otherwise any value must be in the selected set.
function matchesZeroOrManyValuesFilter(values: string[] | undefined, selected: string[]): boolean {
  if (selected.length === 0) return true;
  if (!values || values.length === 0) return false;
  return selected.some((tag) => values.includes(tag));
}

// Check if a map item matches all active map-specific filters.
// Non-map items always match (filters only apply to maps).
function matchesMapAttributeFilters(
  item: InstalledTaggedItem,
  filters: {
    locations: string[];
    sourceQuality: string[];
    levelOfDetail: string[];
    specialDemand: string[];
  },
): boolean {
  if (item.type !== "map") return true;

  const map = item.item as types.MapManifest;
  return (
    matchesSingleValueFilter(map.location, filters.locations) &&
    matchesSingleValueFilter(map.source_quality, filters.sourceQuality) &&
    matchesSingleValueFilter(map.level_of_detail, filters.levelOfDetail) &&
    matchesZeroOrManyValuesFilter(map.special_demand, filters.specialDemand)
  );
}

function buildSearchText(entry: InstalledTaggedItem): string {
  const base = entry.item;
  const values: string[] = [base.name ?? "", base.author ?? ""];

  if (entry.type === "mod") {
    values.push(...(base.tags ?? []));
  } else {
    const map = base as types.MapManifest;
    values.push(
      map.city_code ?? "",
      map.country ?? "",
      map.location ?? "",
      map.source_quality ?? "",
      map.level_of_detail ?? "",
      ...(map.special_demand ?? []),
    );
  }

  return values.filter(Boolean).join(" ");
}

function compareItems(
  a: InstalledTaggedItem,
  b: InstalledTaggedItem,
  sort: LibrarySortOption,
): number {
  const countryA = a.type === "map" ? ((a.item as types.MapManifest).country ?? "") : "";
  const countryB = b.type === "map" ? ((b.item as types.MapManifest).country ?? "") : "";

  switch (sort) {
    case "name-asc":
      return (a.item.name ?? "").localeCompare(b.item.name ?? "");
    case "name-desc":
      return (b.item.name ?? "").localeCompare(a.item.name ?? "");
    case "author-asc":
      return (a.item.author ?? "").localeCompare(b.item.author ?? "");
    case "country-asc":
      return countryA.localeCompare(countryB) || (a.item.name ?? "").localeCompare(b.item.name ?? "");
    case "country-desc":
      return countryB.localeCompare(countryA) || (a.item.name ?? "").localeCompare(b.item.name ?? "");
    default:
      return 0;
  }
}

export function useFilteredInstalledItems({
  items,
}: UseFilteredInstalledParams) {
  const defaultPerPage = useProfileStore((s) => s.defaultPerPage)() as PerPage;
  const filters = useLibraryStore((s) => s.filters);
  const setFilters = useLibraryStore((s) => s.setFilters);
  const page = useLibraryStore((s) => s.page);
  const setPage = useLibraryStore((s) => s.setPage);

  // Sync perPage with user profile default
  useEffect(() => {
    setFilters((prev) =>
      prev.perPage === defaultPerPage ? prev : { ...prev, perPage: defaultPerPage },
    );
  }, [defaultPerPage, setFilters]);

  // Reset to page 1 when filters change
  const didMount = useRef(false);
  useEffect(() => {
    if (!didMount.current) {
      didMount.current = true;
      return;
    }
    setPage(1);
  }, [filters, setPage]);

  const filtered = useMemo(() => {
    let result = [...items];

    // Filter by active library type
    result = result.filter((i) => i.type === filters.type);

    // Filter by mod tags
    if (filters.type === "mod" && filters.mod.tags.length > 0) {
      result = result.filter((i) =>
        i.type === "mod"
          ? (i.item.tags ?? []).some((tag) => filters.mod.tags.includes(tag))
          : true,
      );
    }

    result = result.filter((i) => matchesMapAttributeFilters(i, filters.map));

    // Search
    const query = filters.query.trim();
    if (query) {
      const searchable: SearchableItem[] = result.map((entry) => ({
        entry,
        searchText: buildSearchText(entry),
      }));
      const fuse = new Fuse(searchable, FUSE_SEARCH_OPTIONS);
      result = fuse.search(query).map((r: { item: SearchableItem }) => r.item.entry);
    }

    // Sort
    return result.sort((a, b) => compareItems(a, b, filters.sort));
  }, [items, filters]);

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
    setPage,
  };
}
