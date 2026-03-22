import { Inbox, Plus } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'wouter';

import { ImportAssetDialog } from '@/components/dialogs/ImportAssetDialog';
import { LibraryActionBar } from '@/components/library/LibraryActionBar';
import { LibrarySidebar } from '@/components/library/LibrarySidebar';
import { LibraryTable } from '@/components/library/LibraryTable';
import { SearchBar } from '@/components/search/SearchBar';
import { EmptyState } from '@/components/shared/EmptyState';
import { ErrorBanner } from '@/components/shared/ErrorBanner';
import { PageHeading } from '@/components/shared/PageHeading';
import { Pagination } from '@/components/shared/Pagination';
import { Button } from '@/components/ui/button';
import { useFilteredInstalledItems } from '@/hooks/use-filtered-installed-items';
import { buildAssetListingCounts } from '@/lib/listing-counts';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { buildSpecialDemandValues } from '@/lib/map-filter-values';
import {
  indexPendingSubscriptionUpdates,
  type PendingUpdatesByKey,
  requestLatestSubscriptionUpdatesForActiveProfile,
} from '@/lib/subscription-updates';
import { useInstalledStore } from '@/stores/installed-store';
import { useRegistryStore } from '@/stores/registry-store';

import { types } from '../../wailsjs/go/models';

function localMapManifestFromInstalled(
  installed: types.InstalledMapInfo,
): types.MapManifest | null {
  const config = installed.config;
  if (!config || !config.code) {
    return null;
  }

  return new types.MapManifest({
    schema_version: 1,
    id: installed.id,
    name: config.name,
    author: config.creator,
    github_id: 0,
    last_updated: 0,
    city_code: config.code,
    country: config.country,
    location: '',
    population: config.population,
    description: config.description,
    data_source: '',
    source_quality: '',
    level_of_detail: '',
    special_demand: [],
    initial_view_state: config.initialViewState || {},
    tags: [],
    gallery: [],
    source: '',
    update: {
      type: 'local',
    },
  });
}

const INSTALL_ACCENT = getLocalAccentClasses('install');
const IMPORT_ACCENT = getLocalAccentClasses('import');

export function LibraryPage() {
  const {
    mods,
    maps,
    modDownloadTotals,
    mapDownloadTotals,
    ensureDownloadTotals,
  } = useRegistryStore();
  const { installedMods, installedMaps, updateInstalledLists } =
    useInstalledStore();
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [pendingUpdatesByKey, setPendingUpdatesByKey] =
    useState<PendingUpdatesByKey>({});

  const refreshPendingSubscriptionUpdates = useCallback(async () => {
    let result;
    try {
      result = await requestLatestSubscriptionUpdatesForActiveProfile({
        apply: false,
      });
    } catch (err) {
      setPendingUpdatesByKey({});
      console.warn(
        `[library:latest_check] Failed to resolve pending updates: ${err instanceof Error ? err.message : String(err)}`,
      );
      return;
    }

    if (result.status === 'error') {
      setPendingUpdatesByKey({});
      console.warn(
        `[library:latest_check] Failed to resolve pending updates: ${result.message}`,
      );
      return;
    }

    setPendingUpdatesByKey(
      indexPendingSubscriptionUpdates(result.pendingUpdates),
    );
    if (result.status === 'warn') {
      console.warn(
        `[library:latest_check] Completed with warnings: ${result.message}`,
      );
    }
  }, []);

  useEffect(() => {
    ensureDownloadTotals();
    void refreshPendingSubscriptionUpdates();
  }, [ensureDownloadTotals, refreshPendingSubscriptionUpdates]);

  const modManifestById = useMemo(
    () => new Map(mods.map((manifest) => [manifest.id, manifest])),
    [mods],
  );
  const mapManifestById = useMemo(
    () => new Map(maps.map((manifest) => [manifest.id, manifest])),
    [maps],
  );

  const missingInstalledItems = useMemo(() => {
    const missingMods = installedMods
      .filter(
        (installed) => !installed.isLocal && !modManifestById.has(installed.id),
      )
      .map((installed) => `mod:${installed.id}`);
    const missingMaps = installedMaps
      .filter(
        (installed) => !installed.isLocal && !mapManifestById.has(installed.id),
      )
      .map((installed) => `map:${installed.id}`);

    return [...missingMods, ...missingMaps];
  }, [installedMaps, installedMods, mapManifestById, modManifestById]);

  const installedItems = useMemo(() => {
    const modItems = installedMods.flatMap((installed) => {
      const manifest = modManifestById.get(installed.id);
      return manifest
        ? [
            {
              type: 'mod' as const,
              item: manifest,
              installedVersion: installed.version,
              isLocal: installed.isLocal,
            },
          ]
        : [];
    });
    const mapItems = installedMaps.flatMap((installed) => {
      const manifest = mapManifestById.get(installed.id);
      if (manifest) {
        return [
          {
            type: 'map' as const,
            item: manifest,
            installedVersion: installed.version,
            isLocal: installed.isLocal,
          },
        ];
      }

      if (!installed.isLocal) {
        return [];
      }

      const localManifest = localMapManifestFromInstalled(installed);
      if (!localManifest) {
        return [];
      }

      return [
        {
          type: 'map' as const,
          item: localManifest,
          installedVersion: installed.version,
          isLocal: true,
        },
      ];
    });

    return [...modItems, ...mapItems];
  }, [installedMods, installedMaps, modManifestById, mapManifestById]);

  const {
    items: paginatedItems,
    allFilteredItems,
    page,
    totalPages,
    totalResults,
    filters,
    setFilters,
    setType,
    setPage,
  } = useFilteredInstalledItems({
    items: installedItems,
    modDownloadTotals,
    mapDownloadTotals,
  });

  const modCount = installedItems.filter((i) => i.type === 'mod').length;
  const mapCount = installedItems.filter((i) => i.type === 'map').length;

  const installedModItems = useMemo(
    () =>
      installedItems
        .filter((entry) => entry.type === 'mod')
        .map((entry) => entry.item),
    [installedItems],
  );
  const installedMapItems = useMemo(
    () =>
      installedItems
        .filter((entry) => entry.type === 'map')
        .map((entry) => entry.item),
    [installedItems],
  );

  const availableTags = useMemo(() => {
    const tags = new Set(installedModItems.flatMap((item) => item.tags ?? []));
    return Array.from(tags).sort();
  }, [installedModItems]);

  const availableSpecialDemand = useMemo(
    () => buildSpecialDemandValues(installedMapItems),
    [installedMapItems],
  );

  const {
    modTagCounts,
    mapLocationCounts,
    mapSourceQualityCounts,
    mapLevelOfDetailCounts,
    mapSpecialDemandCounts,
  } = useMemo(
    () => buildAssetListingCounts(installedModItems, installedMapItems),
    [installedMapItems, installedModItems],
  );

  return (
    <div className="space-y-5">
      <PageHeading
        icon={Inbox}
        title="Library"
        description="Manage your installed maps and mods."
      />

      {missingInstalledItems.length > 0 && (
        <ErrorBanner
          message={
            missingInstalledItems.length === 1
              ? `Installed content is missing from the registry: ${missingInstalledItems[0]}`
              : `${missingInstalledItems.length} installed items are missing from the registry.`
          }
        />
      )}

      <div className="flex items-center gap-3">
        <div className="flex-1">
          <SearchBar
            query={filters.query}
            onQueryChange={(value) =>
              setFilters((prev) => ({ ...prev, query: value }))
            }
          />
        </div>
        <Link href="/search">
          <Button className={`gap-1.5 shrink-0 ${INSTALL_ACCENT.solidButton}`}>
            <Plus className="h-4 w-4" />
            Install Content
          </Button>
        </Link>
        <Button
          variant="outline"
          className={`gap-1.5 shrink-0 ${IMPORT_ACCENT.outlineButton}`}
          onClick={() => setImportDialogOpen(true)}
        >
          <Inbox className="h-4 w-4" />
          Import Asset
        </Button>
      </div>

      {installedItems.length === 0 ? (
        <EmptyState
          icon={Inbox}
          title="No content installed"
          description="Your library is empty. Browse the registry to discover and install community content."
        >
          <Link href="/search">
            <Button className={`gap-1.5 ${INSTALL_ACCENT.solidButton}`}>
              <Plus className="h-4 w-4" />
              Install Content
            </Button>
          </Link>
        </EmptyState>
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>
                <span className="font-medium text-foreground">
                  {totalResults}
                </span>{' '}
                result{totalResults !== 1 ? 's' : ''}
                {filters.query && (
                  <span className="ml-1">
                    for <span className="italic">"{filters.query}"</span>
                  </span>
                )}
              </span>
            </div>
          </div>

          <div className="flex gap-6 items-start">
            <aside className="w-52 shrink-0">
              <LibrarySidebar
                filters={filters}
                onFiltersChange={setFilters}
                onTypeChange={setType}
                modCount={modCount}
                mapCount={mapCount}
                availableTags={availableTags}
                availableSpecialDemand={availableSpecialDemand}
                modTagCounts={modTagCounts}
                mapLocationCounts={mapLocationCounts}
                mapSourceQualityCounts={mapSourceQualityCounts}
                mapLevelOfDetailCounts={mapLevelOfDetailCounts}
                mapSpecialDemandCounts={mapSpecialDemandCounts}
              />
            </aside>

            <div className="flex-1 min-w-0 space-y-4">
              {paginatedItems.length === 0 ? (
                <EmptyState
                  icon={Inbox}
                  title={
                    filters.type === 'map' ? 'No maps found' : 'No mods found'
                  }
                  description={
                    filters.query
                      ? `No installed ${filters.type} match "${filters.query}"`
                      : `No installed ${filters.type} match the current filters`
                  }
                  className="py-16"
                />
              ) : (
                <>
                  <LibraryTable
                    items={paginatedItems}
                    activeType={filters.type}
                    pendingUpdatesByKey={pendingUpdatesByKey}
                    onRefreshPendingUpdates={refreshPendingSubscriptionUpdates}
                    sort={filters.sort}
                    onSortChange={(value) =>
                      setFilters((prev) => ({
                        ...prev,
                        sort: value,
                      }))
                    }
                  />
                  <Pagination
                    page={page}
                    totalPages={totalPages}
                    totalResults={totalResults}
                    perPage={filters.perPage}
                    onPageChange={setPage}
                    onPerPageChange={(value) =>
                      setFilters((prev) => ({
                        ...prev,
                        perPage: value,
                      }))
                    }
                  />
                </>
              )}

              <div className="sticky bottom-4">
                <LibraryActionBar
                  allItems={allFilteredItems}
                  pendingUpdatesByKey={pendingUpdatesByKey}
                  onRefreshPendingUpdates={refreshPendingSubscriptionUpdates}
                />
              </div>
            </div>
          </div>
        </>
      )}

      <ImportAssetDialog
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        onImportSuccess={() => {
          void updateInstalledLists();
          void refreshPendingSubscriptionUpdates();
        }}
      />
    </div>
  );
}
