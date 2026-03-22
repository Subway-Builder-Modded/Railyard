import {
  ArrowRight,
  CircleFadingArrowUp,
  Compass,
  Download,
  Inbox,
  History,
  MapPin,
  Package,
  RefreshCw,
  Settings,
  Terminal,
  TrainTrack,
} from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'wouter';

import { DiscoverSectionGrid } from '@/components/homepage/DiscoverSectionGrid';
import { PendingUpdateRow } from '@/components/homepage/PendingUpdateRow';
import { QuickNavCard } from '@/components/homepage/QuickNavCard';
import { SectionHeader } from '@/components/homepage/SectionHeader';
import { PageHeading } from '@/components/shared/PageHeading';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import type { AssetType } from '@/lib/asset-types';
import { getLocalAccentClasses } from '@/lib/local-accent';
import {
  indexPendingSubscriptionUpdates,
  type PendingUpdatesByKey,
  requestLatestSubscriptionUpdatesForActiveProfile,
} from '@/lib/subscription-updates';
import { sortTaggedItemsByLastUpdated } from '@/lib/tagged-items';
import { cn } from '@/lib/utils';
import { useInstalledStore } from '@/stores/installed-store';
import { useRegistryStore } from '@/stores/registry-store';

const UPDATE_ACCENT = getLocalAccentClasses('update');

export function HomePage() {
  const {
    mods,
    maps,
    loading: registryLoading,
    error: registryError,
    modDownloadTotals,
    mapDownloadTotals,
    ensureDownloadTotals,
  } = useRegistryStore();
  const {
    installedMods,
    installedMaps,
    updateAssetsToLatest,
    isOperating,
    getInstalledVersion,
  } = useInstalledStore();

  const [pendingUpdatesByKey, setPendingUpdatesByKey] =
    useState<PendingUpdatesByKey>({});
  const [updatesLoading, setUpdatesLoading] = useState(false);
  const [updatingAll, setUpdatingAll] = useState(false);

  const installedCount = installedMods.length + installedMaps.length;

  const getTotalDownloads = useCallback(
    (type: AssetType, id: string) =>
      type === 'mod' ? (modDownloadTotals[id] ?? 0) : (mapDownloadTotals[id] ?? 0),
    [modDownloadTotals, mapDownloadTotals],
  );

  const fetchPendingUpdates = useCallback(async () => {
    setUpdatesLoading(true);
    try {
      const result = await requestLatestSubscriptionUpdatesForActiveProfile({
        apply: false,
      });
      if (result.status !== 'error') {
        setPendingUpdatesByKey(
          indexPendingSubscriptionUpdates(result.pendingUpdates),
        );
      }
    } catch {
      // Updates are a convenience, not a critical path on the home page.
    } finally {
      setUpdatesLoading(false);
    }
  }, []);

  useEffect(() => {
    if (installedCount > 0) {
      void fetchPendingUpdates();
    }
  }, [installedCount, fetchPendingUpdates]);

  useEffect(() => {
    ensureDownloadTotals();
  }, [ensureDownloadTotals]);

  const modManifestById = useMemo(
    () => new Map(mods.map((mod) => [mod.id, mod])),
    [mods],
  );
  const mapManifestById = useMemo(
    () => new Map(maps.map((map) => [map.id, map])),
    [maps],
  );

  const pendingUpdateEntries = useMemo(
    () =>
      Object.entries(pendingUpdatesByKey).map(([key, update]) => {
        const type = update.type as AssetType;
        const manifest =
          type === 'mod'
            ? modManifestById.get(update.assetId)
            : mapManifestById.get(update.assetId);

        return {
          key,
          id: update.assetId,
          type,
          name: manifest?.name ?? update.assetId,
          currentVersion: update.currentVersion,
          latestVersion: update.latestVersion,
        };
      }),
    [mapManifestById, modManifestById, pendingUpdatesByKey],
  );

  const handleUpdateSingle = useCallback(
    async (type: AssetType, id: string) => {
      try {
        await updateAssetsToLatest([{ type, id }]);
        void fetchPendingUpdates();
      } catch {
        // Errors via toasts in the store.
      }
    },
    [fetchPendingUpdates, updateAssetsToLatest],
  );

  const handleUpdateAll = useCallback(async () => {
    setUpdatingAll(true);
    try {
      await updateAssetsToLatest(
        pendingUpdateEntries.map(({ type, id }) => ({ type, id })),
      );
      void fetchPendingUpdates();
    } catch {
      // Errors via toasts in the store.
    } finally {
      setUpdatingAll(false);
    }
  }, [fetchPendingUpdates, pendingUpdateEntries, updateAssetsToLatest]);

  const recentMaps = useMemo(
    () =>
      sortTaggedItemsByLastUpdated(
        maps.map((item) => ({ type: 'map' as const, item })),
        'desc',
      ).slice(0, 8),
    [maps],
  );

  const recentMods = useMemo(
    () =>
      sortTaggedItemsByLastUpdated(
        mods.map((item) => ({ type: 'mod' as const, item })),
        'desc',
      ).slice(0, 8),
    [mods],
  );

  const hasActiveUpdateOperation = useMemo(
    () =>
      updatingAll ||
      pendingUpdateEntries.some(({ id }) => isOperating(id)),
    [isOperating, pendingUpdateEntries, updatingAll],
  );

  const showEmptyUpdatesState =
    pendingUpdateEntries.length === 0 && !hasActiveUpdateOperation;
  const emptyUpdatesMessage ="Available updates for installed content will appear here.";
  const EmptyUpdatesIcon = CircleFadingArrowUp;

  return (
    <div className="flex flex-col gap-[clamp(1.75rem,3vw,2.5rem)]">
      <PageHeading
        icon={TrainTrack}
        title="Railyard"
        className="mb-6 sm:mb-8 [&_h1]:gap-3.5 [&_h1]:text-6xl sm:[&_h1]:text-7xl [&_h1_svg]:size-[1.05em]"
      />

      <div className="relative z-10 grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="flex flex-col rounded-xl border border-border bg-card p-[clamp(1rem,1.8vw,1.4rem)]">
          <SectionHeader title="Jump Back In" icon={History} />
          <div className="flex flex-col gap-2">
            <QuickNavCard
              href="/search"
              icon={Compass}
              label="Browse"
              description="Discover community-made content"
            />
            <QuickNavCard
              href="/library"
              icon={Inbox}
              label="Library"
              description="View and manage your installed content"
            />
            <QuickNavCard
              href="/logs"
              icon={Terminal}
              label="Logs"
              description="View Subway Builder game logs"
            />
            <QuickNavCard
              href="/settings"
              icon={Settings}
              label="Settings"
              description="Modify Railyard preferences and configurations"
            />
          </div>
        </div>

        <div className="flex flex-col rounded-xl border border-border bg-card p-[clamp(1rem,1.8vw,1.4rem)]">
          <SectionHeader
            title="Available Updates"
            icon={CircleFadingArrowUp}
            badge={
              !updatesLoading && pendingUpdateEntries.length > 0 ? (
                <Badge
                  size="sm"
                  className="border-[color-mix(in_oklab,var(--update-primary)_35%,transparent)] bg-[color-mix(in_oklab,var(--update-primary)_14%,transparent)] text-[var(--update-primary)]"
                >
                  {pendingUpdateEntries.length}
                </Badge>
              ) : undefined
            }
            action={
              !updatesLoading && pendingUpdateEntries.length > 1 ? (
                <Button
                  size="sm"
                  disabled={updatingAll}
                  onClick={handleUpdateAll}
                  className={cn('h-8 gap-1.5 text-xs', UPDATE_ACCENT.solidButton)}
                >
                  {updatingAll ? (
                    <RefreshCw className="h-3 w-3 animate-spin" aria-hidden />
                  ) : (
                    <Download className="h-3 w-3" aria-hidden />
                  )}
                  Update All
                </Button>
              ) : undefined
            }
          />

          <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto">
            {showEmptyUpdatesState ? (
              <div className="flex flex-1 flex-col items-center justify-center gap-2 py-8 text-center">
                <EmptyUpdatesIcon
                  className="h-12 w-12 text-muted-foreground mb-2"
                  aria-hidden
                />
                <p className="text-sm text-muted-foreground">{emptyUpdatesMessage}</p>
              </div>
            ) : updatesLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-[2.6rem] w-full rounded-lg" />
              ))
            ) : (
              pendingUpdateEntries.map(
                ({ key, id, type, name, currentVersion, latestVersion }) => (
                  <PendingUpdateRow
                    key={key}
                    name={name}
                    type={type}
                    currentVersion={currentVersion}
                    latestVersion={latestVersion}
                    isUpdating={isOperating(id)}
                    onUpdate={() => void handleUpdateSingle(type, id)}
                    updateButtonClassName={UPDATE_ACCENT.solidButton}
                  />
                ),
              )
            )}
          </div>
        </div>
      </div>

      <section>
        <SectionHeader
          title="Discover Maps"
          icon={MapPin}
          action={
            <Link href="/search">
              <Button
                variant="ghost"
                size="sm"
                className="h-8 gap-1 text-xs text-muted-foreground hover:bg-accent/45 hover:text-primary"
              >
                View
                <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            </Link>
          }
        />
        <DiscoverSectionGrid
          items={recentMaps}
          getInstalledVersion={getInstalledVersion}
          getTotalDownloads={getTotalDownloads}
          loading={registryLoading}
          error={registryError}
          emptyMessage="No maps in the registry yet."
        />
      </section>

      <section>
        <SectionHeader
          title="Discover Mods"
          icon={Package}
          action={
            <Link href="/search">
              <Button
                variant="ghost"
                size="sm"
                className="h-8 gap-1 text-xs text-muted-foreground hover:bg-accent/45 hover:text-primary"
              >
                View
                <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            </Link>
          }
        />
        <DiscoverSectionGrid
          items={recentMods}
          getInstalledVersion={getInstalledVersion}
          getTotalDownloads={getTotalDownloads}
          loading={registryLoading}
          error={registryError}
          emptyMessage="No mods in the registry yet."
        />
      </section>
    </div>
  );
}
