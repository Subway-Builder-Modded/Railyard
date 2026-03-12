import { useEffect, useMemo } from "react";
import { useRegistryStore } from "@/stores/registry-store";
import { useFilteredItems } from "@/hooks/use-filtered-items";
import { SearchBar } from "@/components/search/SearchBar";
import { SidebarFilters } from "@/components/search/SidebarFilters";
import { ItemCard } from "@/components/shared/ItemCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { CardSkeletonGrid } from "@/components/shared/CardSkeletonGrid";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { Pagination } from "@/components/shared/Pagination";
import { SortSelect } from "@/components/search/SortSelect";
import { SearchX } from "lucide-react";
import { useInstalledStore } from "@/stores/installed-store";
import { createRandomSeed } from "@/stores/search-store";

export function SearchPage() {
  const {
    mods,
    maps,
    loading,
    error,
    modDownloadTotals,
    mapDownloadTotals,
    ensureDownloadTotals,
  } = useRegistryStore();
  const { installedMaps, installedMods } = useInstalledStore();
  const installedItems = useMemo(() => {
    const items: Array<{
      type: "mods" | "maps";
      item: (typeof mods)[number] | (typeof maps)[number];
      installedVersion: string;
    }> = [];
    for (const installed of installedMods) {
      const manifest = mods.find((m) => m.id === installed.id);
      if (manifest)
        items.push({
          type: "mods",
          item: manifest,
          installedVersion: installed.version,
        });
    }
    for (const installed of installedMaps) {
      const manifest = maps.find((m) => m.id === installed.id);
      if (manifest)
        items.push({
          type: "maps",
          item: manifest,
          installedVersion: installed.version,
        });
    }
    return items;
  }, [mods, maps, installedMods, installedMaps]);

  const allTags = useMemo(() => {
    const modTags = mods.flatMap((m) => m.tags ?? []);
    return [...new Set(modTags)].sort();
  }, [mods]);

  const specialDemandTags = useMemo(() => {
    const dynamicTags = maps.flatMap((m) => m.special_demand ?? []);
    return [...new Set(dynamicTags)].sort();
  }, [maps]);

  useEffect(() => {
    ensureDownloadTotals();
  }, [ensureDownloadTotals]);

  const {
    items,
    page,
    totalPages,
    totalResults,
    filters,
    setFilters,
    setPage,
  } = useFilteredItems({
    mods,
    maps,
    modDownloadTotals,
    mapDownloadTotals,
  });

  const modCount = mods.length;
  const mapCount = maps.length;

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground text-balance">
          Browse
        </h1>
        <p className="text-sm text-muted-foreground mt-0.5">
          Discover and install maps and mods for Subway Builder.
        </p>
      </div>

      {error && <ErrorBanner message={error} />}

      {/* Search bar - full width at top */}
      <SearchBar
        query={filters.query}
        onQueryChange={(value) =>
          setFilters((prev) => ({ ...prev, query: value }))
        }
      />

      {/* Two-column layout: sidebar + results */}
      <div className="flex gap-6 items-start">
        {/* Sidebar */}
        <aside className="w-52 shrink-0">
          <SidebarFilters
            filters={filters}
            onFiltersChange={setFilters}
            availableTags={allTags}
            availableSpecialDemand={specialDemandTags}
            modCount={modCount}
            mapCount={mapCount}
          />
        </aside>

        {/* Main results area */}
        <div className="flex-1 min-w-0 space-y-4">
          {/* Results toolbar */}
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              {loading ? (
                <span className="inline-block h-4 w-24 bg-muted rounded animate-pulse" />
              ) : (
                <>
                  <span className="font-medium text-foreground">
                    {totalResults}
                  </span>{" "}
                  result{totalResults !== 1 ? "s" : ""}
                  {filters.query && (
                    <span className="ml-1">
                      for <span className="italic">"{filters.query}"</span>
                    </span>
                  )}
                </>
              )}
            </p>
            <SortSelect
              value={filters.sort}
              onChange={(value) =>
                setFilters((prev) => ({
                  ...prev,
                  sort: value,
                  randomSeed:
                    value.field === "random" ? createRandomSeed() : prev.randomSeed,
                }))
              }
              tab={filters.type}
            />
          </div>

          {/* Cards / empty / loading */}
          {loading ? (
            <CardSkeletonGrid count={filters.perPage} />
          ) : items.length === 0 ? (
            <EmptyState
              icon={SearchX}
              title="No results found"
              description={
                filters.query
                  ? `No items match "${filters.query}"`
                  : "No items match the current filters"
              }
            />
          ) : (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                {items.map(({ type: itemType, item }) => (
                  <ItemCard
                    key={`${itemType}-${item.id}`}
                    type={itemType}
                    item={item}
                    installedVersion={
                      installedItems.find((i) => i.item === item)
                        ?.installedVersion
                    }
                    totalDownloads={
                      itemType === "mods"
                        ? (modDownloadTotals[item.id] ?? 0)
                        : (mapDownloadTotals[item.id] ?? 0)
                    }
                  />
                ))}
              </div>
              <Pagination
                page={page}
                totalPages={totalPages}
                totalResults={totalResults}
                perPage={filters.perPage}
                onPageChange={setPage}
                onPerPageChange={(value) =>
                  setFilters((prev) => ({ ...prev, perPage: value }))
                }
              />
            </>
          )}
        </div>
      </div>
    </div>
  );
}
