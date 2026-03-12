import { useMemo, useEffect, useRef } from "react";
import Fuse from "fuse.js";
import { types } from "../../wailsjs/go/models";
import {
  type PerPage,
  type SortDirection,
  type SortField,
  type SortState,
} from "../lib/constants";
import { FUSE_SEARCH_OPTIONS } from "@/lib/search";
import { useProfileStore } from "@/stores/profile-store";
import { type SearchFilterState, useSearchStore } from "@/stores/search-store";

export type TaggedItem =
  | { type: "mods"; item: types.ModManifest }
  | { type: "maps"; item: types.MapManifest };

export type { TypeFilter, SearchFilterState } from "@/stores/search-store";

interface UseFilteredItemsParams {
  mods: types.ModManifest[];
  maps: types.MapManifest[];
  modDownloadTotals: Record<string, number>;
  mapDownloadTotals: Record<string, number>;
}

type SearchableItem = {
  entry: TaggedItem;
  searchText: string;
};

function buildSearchText(item: TaggedItem): string {
  const base = item.item;
  const values: string[] = [base.name ?? "", base.author ?? "", base.description ?? ""];

  if (item.type === "mods") {
    values.push(...(base.tags ?? []));
  } else {
    const map = base as types.MapManifest;
    values.push(
      map.city_code ?? "",
      map.country ?? "",
      map.location ?? "",
      map.source_quality ?? "",
      map.level_of_detail ?? "",
      ...(map.special_demand ?? [])
    );
  }

  return values.filter(Boolean).join(" ");
}

function matchesSingleValueFilter(value: string | undefined, selected: string[]): boolean {
  if (selected.length === 0) return true;
  if (!value) return false;
  return selected.includes(value);
}

function matchesZeroOrManyValuesFilter(values: string[] | undefined, selected: string[]): boolean {
  if (selected.length === 0) return true;
  if (!values || values.length === 0) return false;
  return selected.some((tag) => values.includes(tag));
}

function matchesMapAttributeFilters(item: TaggedItem, filters: SearchFilterState["map"]): boolean {
  if (item.type !== "maps") return true;

  const map = item.item as types.MapManifest;
  return (
    matchesSingleValueFilter(map.location, filters.locations) &&
    matchesSingleValueFilter(map.source_quality, filters.sourceQuality) &&
    matchesSingleValueFilter(map.level_of_detail, filters.levelOfDetail) &&
    matchesZeroOrManyValuesFilter(map.special_demand, filters.specialDemand)
  );
}

function compareByDirection(a: number, b: number, direction: SortDirection): number {
  return direction === "asc" ? a - b : b - a;
}

function getTotalDownloads(
  item: TaggedItem,
  modDownloadTotals: Record<string, number>,
  mapDownloadTotals: Record<string, number>
): number {
  return item.type === "mods"
    ? modDownloadTotals[item.item.id] ?? 0
    : mapDownloadTotals[item.item.id] ?? 0;
}

// Helper to determine comparation logic based on sort field and direction
function compareItems(
  a: TaggedItem,
  b: TaggedItem,
  sort: SortState,
  modDownloadTotals: Record<string, number>,
  mapDownloadTotals: Record<string, number>
): number {
  const compareText = (left: string, right: string, direction: SortDirection) =>
    direction === "asc" ? left.localeCompare(right) : right.localeCompare(left);

  const compareField = (field: SortField): number => {
    switch (field) {
      case "name":
        return compareText(a.item.name ?? "", b.item.name ?? "", sort.direction);
      case "author":
        return compareText(a.item.author ?? "", b.item.author ?? "", sort.direction);
      case "population": {
        const popA = a.type === "maps" ? (a.item as types.MapManifest).population ?? 0 : 0;
        const popB = b.type === "maps" ? (b.item as types.MapManifest).population ?? 0 : 0;
        return compareByDirection(popA, popB, sort.direction);
      }
      case "downloads": {
        const downloadsA = getTotalDownloads(a, modDownloadTotals, mapDownloadTotals);
        const downloadsB = getTotalDownloads(b, modDownloadTotals, mapDownloadTotals);
        return compareByDirection(downloadsA, downloadsB, sort.direction);
      }
      default:
        return 0;
    }
  };

  return compareField(sort.field);
}

// Seeded hash function to provide consistent "random" sorting. Stable across renders, but different across sessions
function seededHash(value: string, seed: number): number {
  const FNV_OFFSET_BASIS_32 = 0x811c9dc5;
  const FNV_PRIME_32 = 0x01000193;

  let hash = (seed ^ FNV_OFFSET_BASIS_32) >>> 0;
  for (let i = 0; i < value.length; i += 1) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, FNV_PRIME_32) >>> 0;
  }
  return hash;
}

function sortItemsBySeed(items: TaggedItem[], seed: number): TaggedItem[] {
  return [...items].sort((a, b) => {
    const hashA = seededHash(`${a.type}:${a.item.id}`, seed);
    const hashB = seededHash(`${b.type}:${b.item.id}`, seed);
    if (hashA !== hashB) {
      return hashA - hashB;
    }
    return a.item.id.localeCompare(b.item.id);
  });
}

export function useFilteredItems({
  mods,
  maps,
  modDownloadTotals,
  mapDownloadTotals,
}: UseFilteredItemsParams) {
  const defaultPerPage = useProfileStore((s) => s.defaultPerPage)() as PerPage;
  const filters = useSearchStore((s) => s.filters);
  const setFilters = useSearchStore((s) => s.setFilters);
  const page = useSearchStore((s) => s.page);
  const setPage = useSearchStore((s) => s.setPage);

  useEffect(() => {
    setFilters((prev) =>
      prev.perPage === defaultPerPage
        ? prev
        : {
          ...prev,
          perPage: defaultPerPage,
        }
    );
  }, [defaultPerPage, setFilters]);

  const didMount = useRef(false);
  useEffect(() => {
    if (!didMount.current) {
      didMount.current = true;
      return;
    }
    setPage(1);
  }, [filters, setPage]);

  const allItems = useMemo<TaggedItem[]>(() => {
    const modItems: TaggedItem[] = (mods || []).map((m) => ({ type: "mods" as const, item: m }));
    const mapItems: TaggedItem[] = (maps || []).map((m) => ({ type: "maps" as const, item: m }));
    return [...modItems, ...mapItems];
  }, [mods, maps]);

  const filtered = useMemo(() => {
    let result = allItems.filter((i) => i.type === filters.type);

    if (filters.mod.tags.length > 0) {
      result = result.filter((i) =>
        i.type === "mods" ? matchesZeroOrManyValuesFilter(i.item.tags, filters.mod.tags) : true
      );
    }

    result = result.filter((i) => matchesMapAttributeFilters(i, filters.map));
    const query = filters.query.trim();
    if (query) {
      const searchable: SearchableItem[] = result.map((entry) => ({
        entry,
        searchText: buildSearchText(entry),
      }));

      const fuse = new Fuse(searchable, FUSE_SEARCH_OPTIONS);

      result = fuse.search(query).map(({ item }) => item.entry);
    }

    if (filters.sort.field === "random") {
      return sortItemsBySeed(result, filters.randomSeed);
    }

    return [...result].sort((a, b) =>
      compareItems(a, b, filters.sort, modDownloadTotals, mapDownloadTotals)
    );
  }, [allItems, filters, mapDownloadTotals, modDownloadTotals]);

  const totalResults = filtered.length;
  const totalPages = Math.max(1, Math.ceil(totalResults / filters.perPage));

  const items = useMemo(() => {
    const start = (page - 1) * filters.perPage;
    return filtered.slice(start, start + filters.perPage);
  }, [filtered, page, filters.perPage]);

  return {
    items,
    page,
    totalPages,
    totalResults,
    filters,
    setFilters,
    setPage,
  };
}
