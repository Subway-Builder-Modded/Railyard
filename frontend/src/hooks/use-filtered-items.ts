import { useMemo, useEffect, useRef } from "react";
import { types } from "../../wailsjs/go/models";
import { type PerPage, type SortOption } from "../lib/constants";
import { useProfileStore } from "@/stores/profile-store";
import { type SearchFilterState, useSearchStore } from "@/stores/search-store";

export type TaggedItem =
  | { type: "mods"; item: types.ModManifest }
  | { type: "maps"; item: types.MapManifest };

export type { TypeFilter, SearchFilterState } from "@/stores/search-store";

interface UseFilteredItemsParams {
  mods: types.ModManifest[];
  maps: types.MapManifest[];
}

function matchesQuery(item: TaggedItem, query: string): boolean {
  const q = query.toLowerCase();

  const base = item.item;
  if (base.name?.toLowerCase().includes(q)) return true;
  if (base.author?.toLowerCase().includes(q)) return true;
  if (base.description?.toLowerCase().includes(q)) return true;
  if (item.type === "mods" && base.tags?.some((t) => t.toLowerCase().includes(q))) return true;

  if (item.type === "maps") {
    const map = item.item as types.MapManifest;
    if (map.city_code?.toLowerCase().includes(q)) return true;
    if (map.country?.toLowerCase().includes(q)) return true;
    if (map.location?.toLowerCase().includes(q)) return true;
    if (map.source_quality?.toLowerCase().includes(q)) return true;
    if (map.level_of_detail?.toLowerCase().includes(q)) return true;
    if (map.special_demand?.some((tag) => tag.toLowerCase().includes(q))) return true;
  }

  return false;
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

function compareItems(a: TaggedItem, b: TaggedItem, sort: SortOption): number {
  switch (sort) {
    case "name-asc":
      return (a.item.name ?? "").localeCompare(b.item.name ?? "");
    case "name-desc":
      return (b.item.name ?? "").localeCompare(a.item.name ?? "");
    case "author-asc":
      return (a.item.author ?? "").localeCompare(b.item.author ?? "");
    case "population-desc": {
      const popA = a.type === "maps" ? (a.item as types.MapManifest).population ?? 0 : -1;
      const popB = b.type === "maps" ? (b.item as types.MapManifest).population ?? 0 : -1;
      return popB - popA;
    }
    default:
      return 0;
  }
}

export function useFilteredItems({ mods, maps }: UseFilteredItemsParams) {
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

    if (filters.query.trim()) {
      result = result.filter((i) => matchesQuery(i, filters.query.trim()));
    }

    if (filters.mod.tags.length > 0) {
      result = result.filter((i) =>
        i.type === "mods" ? matchesZeroOrManyValuesFilter(i.item.tags, filters.mod.tags) : true
      );
    }

    result = result.filter((i) => matchesMapAttributeFilters(i, filters.map));

    return [...result].sort((a, b) => compareItems(a, b, filters.sort));
  }, [allItems, filters]);

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
