import { useMemo, useEffect, useRef } from "react";
import Fuse from "fuse.js";
import { types } from "../../wailsjs/go/models";
import { type PerPage } from "../lib/constants";
import { FUSE_SEARCH_OPTIONS } from "@/lib/search";
import { useProfileStore } from "@/stores/profile-store";
import { useLibraryStore } from "@/stores/library-store";

export type InstalledTaggedItem =
  | {
      type: "mods";
      item: types.ModManifest;
      installedVersion: string;
    }
  | {
      type: "maps";
      item: types.MapManifest;
      installedVersion: string;
    };

interface UseFilteredInstalledParams {
  items: InstalledTaggedItem[];
}

type SearchableItem = {
  entry: InstalledTaggedItem;
  searchText: string;
};

function buildSearchText(entry: InstalledTaggedItem): string {
  const base = entry.item;
  const values: string[] = [
    base.name ?? "",
    base.author ?? "",
    base.description ?? "",
  ];

  if (entry.type === "mods") {
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
  sort: string,
): number {
  switch (sort) {
    case "name-asc":
      return (a.item.name ?? "").localeCompare(b.item.name ?? "");
    case "name-desc":
      return (b.item.name ?? "").localeCompare(a.item.name ?? "");
    case "author-asc":
      return (a.item.author ?? "").localeCompare(b.item.author ?? "");
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

    // Filter by type
    if (filters.type !== "all") {
      result = result.filter((i) => i.type === filters.type);
    }

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
