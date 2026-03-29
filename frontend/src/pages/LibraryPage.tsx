import { AlertTriangle, FileArchive, Inbox, Plus, SearchX } from 'lucide-react';
import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { useLocation } from 'wouter';

import { AppDialog } from '@/components/dialogs/AppDialog';
import { LibraryActionBar } from '@/components/library/LibraryActionBar';
import { LibraryList } from '@/components/library/LibraryList';
import {
  LIBRARY_SIDEBAR_CONTENT_OFFSET,
  LibrarySidebarPanel,
} from '@/components/library/LibrarySidebarPanel';
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
  handleSubscriptionMutationError,
  useSubscriptionMutationLockState,
  withLockAwareConfirm,
} from '@/lib/subscription-mutation-ui';
import {
  indexPendingSubscriptionUpdates,
  type PendingUpdatesByKey,
  requestLatestSubscriptionUpdatesForActiveProfile,
} from '@/lib/subscription-updates';
import { useBrowseStore } from '@/stores/browse-store';
import {
  AssetConflictError,
  InvalidMapCodeError,
  useInstalledStore,
} from '@/stores/installed-store';
import { useRegistryStore } from '@/stores/registry-store';
import { useUIStore } from '@/stores/ui-store';

import { OpenImportAssetDialog } from '../../wailsjs/go/main/App';
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
    update: { type: 'local' },
  });
}

function conflictSourceLabel(conflict: types.MapCodeConflict): string {
  if (conflict.existingAssetId?.startsWith('vanilla:')) return 'Vanilla';
  return conflict.existingIsLocal ? 'Local' : 'Registry';
}

function renderPathWithSoftBreaks(path: string) {
  // We want strict bounds in the dialog without breaking mid-segment.
  // Insert optional break points after path separators.
  const parts = path.split(/([\\/])/g);
  return parts.map((part, idx) => {
    if (part === '/' || part === '\\') {
      return (
        <Fragment key={`${idx}-sep`}>
          {part}
          <wbr />
        </Fragment>
      );
    }
    return <Fragment key={`${idx}-txt`}>{part}</Fragment>;
  });
}

const INSTALL_ACCENT = getLocalAccentClasses('install');
const IMPORT_ACCENT = getLocalAccentClasses('import');
const FILES_ACCENT = getLocalAccentClasses('files');

export function LibraryPage() {
  const [, navigate] = useLocation();
  const { locked: mutationLocked, reason: mutationLockedReason } =
    useSubscriptionMutationLockState();
  const sidebarOpen = useUIStore((s) => s.librarySidebarOpen);
  const setSidebarOpen = useUIStore((s) => s.setLibrarySidebarOpen);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [importLoading, setImportLoading] = useState(false);
  const [importSelectedPath, setImportSelectedPath] = useState('');
  const [importConflict, setImportConflict] =
    useState<types.MapCodeConflict | null>(null);
  const [importInvalidCode, setImportInvalidCode] = useState<string | null>(
    null,
  );
  const [pendingUpdatesByKey, setPendingUpdatesByKey] =
    useState<PendingUpdatesByKey>({});

  const {
    mods,
    maps,
    modDownloadTotals,
    mapDownloadTotals,
    ensureDownloadTotals,
  } = useRegistryStore();
  const {
    installedMods,
    installedMaps,
    updateInstalledLists,
    importMapFromZip,
  } = useInstalledStore();

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

  const handleInstallBrowse = useCallback(() => {
    useBrowseStore.getState().setType(filters.type);
    navigate('/browse');
  }, [filters.type, navigate]);

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

  const runImport = async (zipPath: string, replaceOnConflict: boolean) => {
    setImportLoading(true);
    try {
      const result = await importMapFromZip(zipPath, replaceOnConflict);
      if (result.status === 'warn') {
        toast.warning(result.message || 'Map imported with warnings.');
      } else {
        toast.success(result.message || 'Map imported successfully.');
      }
      void updateInstalledLists();
      void refreshPendingSubscriptionUpdates();
      setImportConflict(null);
      setImportSelectedPath('');
      setImportDialogOpen(false);
    } catch (err) {
      if (err instanceof AssetConflictError && err.conflicts.length > 0) {
        setImportConflict(err.conflicts[0]);
        return;
      }
      if (err instanceof InvalidMapCodeError) {
        setImportInvalidCode(err.message);
        return;
      }
      if (handleSubscriptionMutationError(err, () => {})) {
        return;
      }
      setImportSelectedPath('');
      toast.error('Failed to import map.');
    } finally {
      setImportLoading(false);
    }
  };

  const handlePickArchive = async () => {
    if (importLoading) return;
    setImportLoading(true);
    try {
      const selection = await OpenImportAssetDialog('map');
      if (selection.status === 'error') {
        toast.error('Failed to import map.');
        return;
      }
      if (selection.status === 'warn' || !selection.path?.trim()) {
        return;
      }
      setImportSelectedPath(selection.path);
      await runImport(selection.path, false);
    } finally {
      setImportLoading(false);
    }
  };

  return (
    <>
      <LibrarySidebarPanel
        open={sidebarOpen}
        onToggle={() => setSidebarOpen(!sidebarOpen)}
        filters={filters}
        onFiltersChange={setFilters}
        onTypeChange={setType}
        availableTags={availableTags}
        availableSpecialDemand={availableSpecialDemand}
        modTagCounts={modTagCounts}
        mapLocationCounts={mapLocationCounts}
        mapSourceQualityCounts={mapSourceQualityCounts}
        mapLevelOfDetailCounts={mapLevelOfDetailCounts}
        mapSpecialDemandCounts={mapSpecialDemandCounts}
        modCount={modCount}
        mapCount={mapCount}
      />

      <div
        className="space-y-5"
        style={{
          paddingLeft: sidebarOpen ? LIBRARY_SIDEBAR_CONTENT_OFFSET : '0px',
          transition: 'padding-left 200ms ease-out',
          minHeight: 'calc(100vh - var(--app-navbar-offset))',
        }}
      >
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
          <Button
            className={`shrink-0 gap-1.5 ${INSTALL_ACCENT.solidButton}`}
            onClick={handleInstallBrowse}
          >
            <Plus className="h-4 w-4" />
            {filters.type === 'map' ? 'Install Maps' : 'Install Mods'}
          </Button>
          <Button
            variant="outline"
            className={`shrink-0 gap-1.5 ${IMPORT_ACCENT.outlineButton}`}
            onClick={() => setImportDialogOpen(true)}
            disabled={mutationLocked}
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
            <Button
              className={`gap-1.5 ${INSTALL_ACCENT.solidButton}`}
              onClick={handleInstallBrowse}
            >
              <Plus className="h-4 w-4" />
              {filters.type === 'map' ? 'Install Maps' : 'Install Mods'}
            </Button>
          </EmptyState>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              <span className="font-medium text-foreground">
                {totalResults}
              </span>{' '}
              result{totalResults !== 1 ? 's' : ''}
              {filters.query && (
                <span className="ml-1">
                  for <span className="italic">"{filters.query}"</span>
                </span>
              )}
            </p>

            {paginatedItems.length === 0 ? (
              <EmptyState
                icon={SearchX}
                title={
                  filters.type === 'map' ? 'No maps found' : 'No mods found'
                }
                description={
                  filters.query
                    ? `No installed ${filters.type} match "${filters.query}"`
                    : `No installed ${filters.type} match the current filters`
                }
              />
            ) : (
              <>
                <LibraryList
                  items={paginatedItems}
                  activeType={filters.type}
                  pendingUpdatesByKey={pendingUpdatesByKey}
                  onRefreshPendingUpdates={refreshPendingSubscriptionUpdates}
                  sort={filters.sort}
                  onSortChange={(value) =>
                    setFilters((prev) => ({ ...prev, sort: value }))
                  }
                />
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

            <div className="sticky bottom-4">
              <LibraryActionBar
                allItems={allFilteredItems}
                pendingUpdatesByKey={pendingUpdatesByKey}
                onRefreshPendingUpdates={refreshPendingSubscriptionUpdates}
              />
            </div>
          </div>
        )}
      </div>

      <AppDialog
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        title="Import"
        icon={FileArchive}
        description="Import a local map ZIP into your Library. Local assets are tracked separately from registry assets."
        tone="import"
        confirm={withLockAwareConfirm({
          label: 'Choose ZIP',
          cancelLabel: 'Close',
          onConfirm: handlePickArchive,
          loading: importLoading,
        }, mutationLocked, mutationLockedReason)}
      >
        <div className="min-w-0 rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
          Asset Type: <span className="font-medium text-foreground">Map</span>
          {importSelectedPath ? (
            <p className="mt-1 min-w-0 max-w-full overflow-hidden whitespace-normal">
              Selected Archive:{' '}
              <span className="text-foreground font-mono">
                {renderPathWithSoftBreaks(importSelectedPath)}
              </span>
            </p>
          ) : null}
        </div>
      </AppDialog>

      {importConflict && (
        <AppDialog
          open={!!importConflict}
          onOpenChange={(value) => {
            if (!value) setImportConflict(null);
          }}
          title="Replace Conflicting Map"
          icon={AlertTriangle}
          description="This local import conflicts with an existing map. Replace the existing map to continue."
          tone="files"
          confirm={withLockAwareConfirm({
            label: 'Replace',
            onConfirm: () => {
              if (!importSelectedPath) return;
              void runImport(importSelectedPath, true);
            },
            loading: importLoading,
          }, mutationLocked, mutationLockedReason)}
        >
          <div
            className={`rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground ${FILES_ACCENT.dialogPanel}`}
          >
            <p className="font-medium text-foreground">
              Conflicting City Code: {importConflict.cityCode}
            </p>
            <p className="mt-1">
              Existing Asset: {importConflict.existingAssetId} (
              {conflictSourceLabel(importConflict)})
            </p>
            {importConflict.existingVersion ? (
              <p className="mt-1">
                Existing Version: {importConflict.existingVersion}
              </p>
            ) : null}
          </div>
        </AppDialog>
      )}

      {importInvalidCode && (
        <AppDialog
          open={!!importInvalidCode}
          onOpenChange={(value) => {
            if (!value) setImportInvalidCode(null);
          }}
          title="Invalid Local Map Code"
          icon={AlertTriangle}
          description={`${importInvalidCode} Local map codes must be 2-4 uppercase letters (e.g. "AAA").`}
          tone="files"
          confirm={{ label: 'OK', onConfirm: () => setImportInvalidCode(null) }}
        />
      )}
    </>
  );
}
