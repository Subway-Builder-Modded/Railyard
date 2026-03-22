import { Compass, SearchX } from 'lucide-react';
import { useEffect, useMemo } from 'react';

import { SearchBar } from '@/components/search/SearchBar';
import { SidebarFilters } from '@/components/search/SidebarFilters';
import { SortSelect } from '@/components/search/SortSelect';
import { ViewModeToggle } from '@/components/search/ViewModeToggle';
import { CardSkeletonGrid } from '@/components/shared/CardSkeletonGrid';
import { EmptyState } from '@/components/shared/EmptyState';
import { ErrorBanner } from '@/components/shared/ErrorBanner';
import { ItemCard } from '@/components/shared/ItemCard';
import { PageHeading } from '@/components/shared/PageHeading';
import { Pagination } from '@/components/shared/Pagination';
import { ResponsiveCardGrid } from '@/components/shared/ResponsiveCardGrid';
import { useFilteredItems } from '@/hooks/use-filtered-items';
import type { AssetType } from '@/lib/asset-types';
import { buildAssetListingCounts } from '@/lib/listing-counts';
import { buildSpecialDemandValues } from '@/lib/map-filter-values';
import { useInstalledStore } from '@/stores/installed-store';
import { useProfileStore } from '@/stores/profile-store';
import { useRegistryStore } from '@/stores/registry-store';
import { createRandomSeed } from '@/stores/search-store';
import { useSearchStore } from '@/stores/search-store';

export function SearchPage() {
  const viewMode = useSearchStore((s) => s.viewMode);
  const setViewMode = useSearchStore((s) => s.setViewMode);
  const initializeViewMode = useSearchStore((s) => s.initializeViewMode);
  const defaultSearchViewMode = useProfileStore((s) => s.searchViewMode)();

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
      type: AssetType;
      item: (typeof mods)[number] | (typeof maps)[number];
      installedVersion: string;
    }> = [];
    for (const installed of installedMods) {
      const manifest = mods.find((m) => m.id === installed.id);
      if (manifest)
        items.push({
          type: 'mod',
          item: manifest,
          installedVersion: installed.version,
        });
    }
    for (const installed of installedMaps) {
      const manifest = maps.find((m) => m.id === installed.id);
      if (manifest)
        items.push({
          type: 'map',
          item: manifest,
          installedVersion: installed.version,
        });
    }
    return items;
  }, [mods, maps, installedMods, installedMaps]);

  const installedVersionByItemKey = useMemo(() => {
    return new Map(
      installedItems.map((entry) => [
        `${entry.type}-${entry.item.id}`,
        entry.installedVersion,
      ]),
    );
  }, [installedItems]);

  const allTags = useMemo(() => {
    const modTags = mods.flatMap((m) => m.tags ?? []);
    return [...new Set(modTags)].sort();
  }, [mods]);

  const availableSpecialDemand = useMemo(
    () => buildSpecialDemandValues(maps),
    [maps],
  );

  const {
    modTagCounts,
    mapLocationCounts,
    mapSourceQualityCounts,
    mapLevelOfDetailCounts,
    mapSpecialDemandCounts,
  } = useMemo(() => buildAssetListingCounts(mods, maps), [mods, maps]);

  useEffect(() => {
    ensureDownloadTotals();
  }, [ensureDownloadTotals]);

  useEffect(() => {
    initializeViewMode(defaultSearchViewMode);
  }, [defaultSearchViewMode, initializeViewMode]);

  const {
    items,
    page,
    totalPages,
    totalResults,
    filters,
    setFilters,
    setType,
    setPage,
  } = useFilteredItems({
    mods,
    maps,
    modDownloadTotals,
    mapDownloadTotals,
  });

  const modCount = mods.length;
  const mapCount = maps.length;

  const cardGridPreset = useMemo(
    () => (viewMode === 'compact' ? 'compact' : 'default'),
    [viewMode],
  );

  return (
    <div className="space-y-5">
      <PageHeading
        icon={Compass}
        title="Browse"
        description="Discover and install maps and mods for Subway Builder."
      />

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
            onTypeChange={setType}
            availableTags={allTags}
            availableSpecialDemand={availableSpecialDemand}
            modTagCounts={modTagCounts}
            mapLocationCounts={mapLocationCounts}
            mapSourceQualityCounts={mapSourceQualityCounts}
            mapLevelOfDetailCounts={mapLevelOfDetailCounts}
            mapSpecialDemandCounts={mapSpecialDemandCounts}
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
                  </span>{' '}
                  result{totalResults !== 1 ? 's' : ''}
                  {filters.query && (
                    <span className="ml-1">
                      for <span className="italic">"{filters.query}"</span>
                    </span>
                  )}
                </>
              )}
            </p>
            <div className="flex items-center gap-2">
              <ViewModeToggle value={viewMode} onChange={setViewMode} />
              <SortSelect
                value={filters.sort}
                onChange={(value) =>
                  setFilters((prev) => ({
                    ...prev,
                    sort: value,
                    randomSeed:
                      value.field === 'random'
                        ? createRandomSeed()
                        : prev.randomSeed,
                  }))
                }
                tab={filters.type}
              />
            </div>
          </div>

          {/* Cards / empty / loading */}
          {loading ? (
            <CardSkeletonGrid count={filters.perPage} preset={cardGridPreset} />
          ) : items.length === 0 ? (
            <EmptyState
              icon={SearchX}
              title="No results found"
              description={
                filters.query
                  ? `No items match "${filters.query}"`
                  : 'No items match the current filters'
              }
            />
          ) : (
            <>
              {viewMode === 'list' ? (
                <div className="space-y-4">
                  {items.map(({ type: itemType, item }) => (
                    <ItemCard
                      key={`${itemType}-${item.id}`}
                      type={itemType}
                      item={item}
                      viewMode={viewMode}
                      installedVersion={installedVersionByItemKey.get(
                        `${itemType}-${item.id}`,
                      )}
                      totalDownloads={
                        itemType === 'mod'
                          ? (modDownloadTotals[item.id] ?? 0)
                          : (mapDownloadTotals[item.id] ?? 0)
                      }
                    />
                  ))}
                </div>
              ) : (
                <ResponsiveCardGrid preset={cardGridPreset}>
                  {items.map(({ type: itemType, item }) => (
                    <ItemCard
                      key={`${itemType}-${item.id}`}
                      type={itemType}
                      item={item}
                      viewMode={viewMode}
                      installedVersion={installedVersionByItemKey.get(
                        `${itemType}-${item.id}`,
                      )}
                      totalDownloads={
                        itemType === 'mod'
                          ? (modDownloadTotals[item.id] ?? 0)
                          : (mapDownloadTotals[item.id] ?? 0)
                      }
                    />
                  ))}
                </ResponsiveCardGrid>
              )}
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
