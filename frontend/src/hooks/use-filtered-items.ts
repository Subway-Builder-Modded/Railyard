import Fuse from 'fuse.js';
import { useEffect, useMemo, useRef } from 'react';

import { FUSE_SEARCH_OPTIONS } from '@/lib/search';
import {
  buildTaggedItems,
  compareItems,
  type TaggedItem,
} from '@/lib/tagged-items';
import { useProfileStore } from '@/stores/profile-store';
import { type SearchFilterState, useSearchStore } from '@/stores/search-store';

import type { types } from '../../wailsjs/go/models';
import { type PerPage, type SortState } from '../lib/constants';

export type { SearchFilterState } from '@/stores/search-store';

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

export interface TaggedItemFilterState {
  query: string;
  type: 'mod' | 'map';
  sort: SortState;
  randomSeed: number;
  mod: {
    tags: string[];
  };
  map: SearchFilterState['map'];
}

export function buildSearchText(item: TaggedItem): string {
  const base = item.item;
  const values: string[] = [base.name ?? '', base.author ?? ''];

  if (item.type === 'map') {
    const map = base as types.MapManifest;
    values.push(map.city_code ?? '', map.country ?? '');
  }

  return values.filter(Boolean).join(' ');
}

export function matchesSingleValueFilter(
  value: string | undefined,
  selected: string[],
): boolean {
  if (selected.length === 0) return true;
  if (!value) return false;
  return selected.includes(value);
}

export function matchesZeroOrManyValuesFilter(
  values: string[] | undefined,
  selected: string[],
): boolean {
  if (selected.length === 0) return true;
  if (!values || values.length === 0) return false;
  return selected.some((tag) => values.includes(tag));
}

export function matchesMapAttributeFilters(
  item: TaggedItem,
  filters: SearchFilterState['map'],
): boolean {
  if (item.type !== 'map') return true;

  const map = item.item as types.MapManifest;
  return (
    matchesSingleValueFilter(map.location, filters.locations) &&
    matchesSingleValueFilter(map.source_quality, filters.sourceQuality) &&
    matchesSingleValueFilter(map.level_of_detail, filters.levelOfDetail) &&
    matchesZeroOrManyValuesFilter(map.special_demand, filters.specialDemand)
  );
}

// Seeded hash function to provide consistent "random" sorting. Stable across renders, but different across sessions
export function seededHash(value: string, seed: number): number {
  const FNV_OFFSET_BASIS_32 = 0x811c9dc5;
  const FNV_PRIME_32 = 0x01000193;

  let hash = (seed ^ FNV_OFFSET_BASIS_32) >>> 0;
  for (let i = 0; i < value.length; i += 1) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, FNV_PRIME_32) >>> 0;
  }
  return hash;
}

export function sortItemsBySeed(
  items: TaggedItem[],
  seed: number,
): TaggedItem[] {
  return [...items].sort((a, b) => {
    const hashA = seededHash(`${a.type}:${a.item.id}`, seed);
    const hashB = seededHash(`${b.type}:${b.item.id}`, seed);
    if (hashA !== hashB) {
      return hashA - hashB;
    }
    return a.item.id.localeCompare(b.item.id);
  });
}

export function filterAndSortTaggedItems<T extends TaggedItem>(
  items: T[],
  filters: TaggedItemFilterState,
  modDownloadTotals: Record<string, number>,
  mapDownloadTotals: Record<string, number>,
): T[] {
  let result = items.filter((i) => i.type === filters.type);

  if (filters.mod.tags.length > 0) {
    result = result.filter((i) =>
      i.type === 'mod'
        ? matchesZeroOrManyValuesFilter(i.item.tags, filters.mod.tags)
        : true,
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

    result = fuse.search(query).map(({ item }) => item.entry as T);
  }

  if (filters.sort.field === 'random') {
    return sortItemsBySeed(result, filters.randomSeed) as T[];
  }

  return [...result].sort((a, b) =>
    compareItems(a, b, filters.sort, modDownloadTotals, mapDownloadTotals),
  );
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
  const setType = useSearchStore((s) => s.setType);
  const page = useSearchStore((s) => s.page);
  const setPage = useSearchStore((s) => s.setPage);

  useEffect(() => {
    setFilters((prev) =>
      prev.perPage === defaultPerPage
        ? prev
        : {
            ...prev,
            perPage: defaultPerPage,
          },
    );
  }, [defaultPerPage, setFilters]);

  const didMount = useRef(false);
  const previousTypeRef = useRef(filters.type);
  useEffect(() => {
    if (!didMount.current) {
      didMount.current = true;
      previousTypeRef.current = filters.type;
      return;
    }
    if (previousTypeRef.current !== filters.type) {
      previousTypeRef.current = filters.type;
      return;
    }
    setPage(1);
  }, [filters, setPage]);

  const allItems = useMemo<TaggedItem[]>(
    () => buildTaggedItems(mods, maps),
    [mods, maps],
  );

  const filtered = useMemo(() => {
    return filterAndSortTaggedItems(
      allItems,
      filters,
      modDownloadTotals,
      mapDownloadTotals,
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
    setType,
    setPage,
  };
}
