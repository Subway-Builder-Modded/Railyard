import {
  CircleFadingArrowUp,
  FolderOpen,
  Globe,
  HardDrive,
  Hash,
  OctagonX,
  Trash2,
  Type,
} from 'lucide-react';
import { useCallback, useState } from 'react';
import { toast } from 'sonner';
import { Link } from 'wouter';

import { AppDialog } from '@/components/dialogs/AppDialog';
import { GalleryImage } from '@/components/shared/GalleryImage';
import { SortableHeaderCell } from '@/components/shared/SortableHeaderCell';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import type { InstalledTaggedItem } from '@/hooks/use-filtered-installed-items';
import type { AssetType } from '@/lib/asset-types';
import { assetTypeToListingPath } from '@/lib/asset-types';
import {
  type SortDirection,
  type SortField,
  type SortState,
  TEXT_SORT_FIELDS,
} from '@/lib/constants';
import { getCountryFlagIcon } from '@/lib/flags';
import { openInstallFolder } from '@/lib/install-path';
import { LOCAL_ACCENTS } from '@/lib/local-accent';
import { formatSourceQuality } from '@/lib/map-filter-values';
import {
  handleSubscriptionMutationError,
  useSubscriptionMutationLockState,
  withLockAwareConfirm,
} from '@/lib/subscription-mutation-ui';
import {
  composeAssetKey,
  getPendingSubscriptionUpdate,
  type PendingUpdatesByKey,
  type PendingUpdateTarget,
} from '@/lib/subscription-updates';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';
import { useInstalledStore } from '@/stores/installed-store';
import { useLibraryStore } from '@/stores/library-store';

import type { types } from '../../../wailsjs/go/models';

const UPDATE_ICON_ACCENT = LOCAL_ACCENTS.update.iconButton;
const FILES_ICON_ACCENT = LOCAL_ACCENTS.files.iconButton;
const UNINSTALL_ICON_ACCENT = LOCAL_ACCENTS.uninstall.iconButton;

const ENTRIES_PREVIEW_LIMIT = 10;

export function LocalBadge({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full border border-amber-400/30 bg-amber-500/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-amber-600 dark:text-amber-400',
        className,
      )}
    >
      <HardDrive className="h-2.5 w-2.5 shrink-0" />
      Local
    </span>
  );
}

const COL = {
  gap: 'gap-3',
  city: 'w-[5.5rem]',
  country: 'w-[9rem]',
  version: 'w-[6rem]',
  actions: 'w-[5.5rem]',
} as const;

export interface LibraryListProps {
  items: InstalledTaggedItem[];
  activeType: AssetType;
  pendingUpdatesByKey: PendingUpdatesByKey;
  onRefreshPendingUpdates: () => Promise<void>;
  sort: SortState;
  onSortChange: (sort: SortState) => void;
}

export function LibraryList({
  items,
  activeType,
  pendingUpdatesByKey,
  onRefreshPendingUpdates,
  sort,
  onSortChange,
}: LibraryListProps) {
  const { selectedIds, selectAll, clearSelection } = useLibraryStore();
  const { locked: mutationLocked, reason: mutationLockedReason } =
    useSubscriptionMutationLockState();
  const showMapColumns = activeType === 'map';

  const [columnDirections, setColumnDirections] = useState<
    Partial<Record<Exclude<SortField, 'random'>, SortDirection>>
  >({});

  const handleColumnSort = useCallback(
    (field: Exclude<SortField, 'random'>) => {
      const direction: SortDirection =
        sort.field === field
          ? sort.direction === 'asc'
            ? 'desc'
            : 'asc'
          : (columnDirections[field] ?? 'asc');
      setColumnDirections((prev) => ({ ...prev, [field]: direction }));
      onSortChange({ field, direction });
    },
    [sort, columnDirections, onSortChange],
  );

  const allKeys = items.map((e) => composeAssetKey(e.type, e.item.id));
  const allSelected =
    items.length > 0 && allKeys.every((k) => selectedIds.has(k));
  const someSelected = !allSelected && allKeys.some((k) => selectedIds.has(k));

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <div
        className={cn(
          'flex items-center border-b border-border bg-muted/20 px-4 py-2',
          COL.gap,
        )}
      >
        <Checkbox
          checked={allSelected ? true : someSelected ? 'indeterminate' : false}
          onCheckedChange={() =>
            allSelected ? clearSelection() : selectAll(allKeys)
          }
          aria-label="Select all"
          className="h-4 w-4 shrink-0"
        />
        <div className="h-9 w-9 shrink-0" aria-hidden />
        <div className="flex-1 min-w-0">
          <SortableHeaderCell
            label="Name"
            field="name"
            icon={Type}
            sort={sort}
            textFields={TEXT_SORT_FIELDS}
            onSort={handleColumnSort}
          />
        </div>
        {showMapColumns && (
          <>
            <div className={cn(COL.city, 'hidden shrink-0 lg:block')}>
              <SortableHeaderCell
                label="City"
                field="city_code"
                icon={Hash}
                sort={sort}
                textFields={TEXT_SORT_FIELDS}
                onSort={handleColumnSort}
              />
            </div>
            <div className={cn(COL.country, 'hidden shrink-0 lg:block')}>
              <SortableHeaderCell
                label="Country"
                field="country"
                icon={Globe}
                sort={sort}
                textFields={TEXT_SORT_FIELDS}
                onSort={handleColumnSort}
              />
            </div>
          </>
        )}
        <div className={cn(COL.version, 'flex shrink-0 items-center')}>
          <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Version
          </span>
        </div>
        <div className={cn(COL.actions, 'shrink-0')} aria-hidden />
      </div>

      <div className="divide-y divide-border/50">
        {items.map((entry) => (
          <LibraryListRow
            key={composeAssetKey(entry.type, entry.item.id)}
            entry={entry}
            showMapColumns={showMapColumns}
            pendingUpdatesByKey={pendingUpdatesByKey}
            onRefreshPendingUpdates={onRefreshPendingUpdates}
            mutationLocked={mutationLocked}
            mutationLockedReason={mutationLockedReason}
          />
        ))}
      </div>
    </div>
  );
}

interface LibraryListRowProps {
  entry: InstalledTaggedItem;
  showMapColumns: boolean;
  pendingUpdatesByKey: PendingUpdatesByKey;
  onRefreshPendingUpdates: () => Promise<void>;
  mutationLocked: boolean;
  mutationLockedReason?: string;
}

function LibraryListRow({
  entry,
  showMapColumns,
  pendingUpdatesByKey,
  onRefreshPendingUpdates,
  mutationLocked,
  mutationLockedReason,
}: LibraryListRowProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [uninstallLoading, setUninstallLoading] = useState(false);
  const [updateOpen, setUpdateOpen] = useState(false);
  const [updateLoading, setUpdateLoading] = useState(false);

  const { selectedIds, toggleSelected, removeSelected } = useLibraryStore();
  const { uninstallAssets, updateAssetsToLatest } = useInstalledStore();
  const metroMakerDataPath = useConfigStore(
    (s) => s.config?.metroMakerDataPath,
  );

  const key = composeAssetKey(entry.type, entry.item.id);
  const isSelected = selectedIds.has(key);
  const isMap = entry.type === 'map';
  const isLocal = entry.isLocal;
  const map = isMap ? (entry.item as types.MapManifest) : null;

  const mapCityCode = map?.city_code?.trim().toUpperCase() ?? '';
  const mapCountry = map?.country?.trim().toUpperCase() ?? '';
  const CountryFlag = isMap ? getCountryFlagIcon(mapCountry) : null;

  const badges = isMap
    ? [
        map?.location,
        formatSourceQuality(map?.source_quality ?? ''),
        map?.level_of_detail,
        ...(map?.special_demand ?? []),
      ].filter((v): v is string => Boolean(v))
    : (entry.item.tags ?? []);

  const pendingUpdate = isLocal
    ? undefined
    : getPendingSubscriptionUpdate(
        pendingUpdatesByKey,
        entry.type,
        entry.item.id,
      );

  const projectHref = `/project/${assetTypeToListingPath(entry.type)}/${entry.item.id}`;

  const visibleBadges = badges.slice(0, 2);
  const overflowCount = badges.length - visibleBadges.length;

  const handleUninstall = async () => {
    setUninstallLoading(true);
    try {
      await uninstallAssets([{ id: entry.item.id, type: entry.type }]);
      toast.success(`${entry.item.name} has been uninstalled.`);
      removeSelected([key]);
      void onRefreshPendingUpdates();
      setUninstallOpen(false);
    } catch (err) {
      handleSubscriptionMutationError(
        err,
        `Failed to uninstall ${entry.item.name}.`,
      );
    } finally {
      setUninstallLoading(false);
    }
  };

  const updateTarget: PendingUpdateTarget | null = pendingUpdate
    ? {
        id: entry.item.id,
        type: entry.type,
        name: entry.item.name,
        currentVersion: pendingUpdate.currentVersion,
        latestVersion: pendingUpdate.latestVersion,
      }
    : null;

  const handleUpdate = async () => {
    if (!updateTarget) return;
    setUpdateLoading(true);
    try {
      await updateAssetsToLatest([
        { id: updateTarget.id, type: updateTarget.type },
      ]);
      toast.success(`${updateTarget.name} has been updated.`);
      void onRefreshPendingUpdates();
      setUpdateOpen(false);
    } catch (err) {
      handleSubscriptionMutationError(
        err,
        `Failed to update ${updateTarget.name}.`,
      );
    } finally {
      setUpdateLoading(false);
    }
  };

  const updateEntries = updateTarget
    ? [
        {
          key: `${updateTarget.type}-${updateTarget.id}`,
          name: updateTarget.name,
          currentVersion: updateTarget.currentVersion,
          latestVersion: updateTarget.latestVersion,
        },
      ]
    : [];
  const previewEntries = updateEntries.slice(0, ENTRIES_PREVIEW_LIMIT);

  return (
    <>
      <article
        className={cn(
          'flex items-center px-4 py-2.5 transition-colors',
          COL.gap,
          'hover:bg-muted/30',
          isSelected && 'bg-primary/[0.04]',
        )}
      >
        <Checkbox
          checked={isSelected}
          onCheckedChange={() => toggleSelected(key)}
          aria-label={`Select ${entry.item.name}`}
          className="h-4 w-4 shrink-0"
        />

        <div className="h-9 w-9 shrink-0 overflow-hidden rounded-lg bg-muted">
          <GalleryImage
            type={entry.type}
            id={entry.item.id}
            imagePath={entry.item.gallery?.[0]}
            className="h-full w-full object-cover"
            fallbackIconClassName="h-4 w-4"
          />
        </div>

        <div className="flex-1 min-w-0 flex items-center gap-2">
          <div className="flex-1 min-w-0">
            {isLocal ? (
              <span className="block truncate text-sm font-semibold leading-snug text-foreground">
                {entry.item.name}
              </span>
            ) : (
              <Link
                href={projectHref}
                className="block truncate text-sm font-semibold leading-snug text-foreground hover:underline"
              >
                {entry.item.name}
              </Link>
            )}
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              by {entry.item.author}
            </p>
          </div>

          <div className="shrink-0 flex items-center gap-1">
            {isLocal ? (
              <LocalBadge />
            ) : (
              <>
                {visibleBadges.map((badge) => (
                  <Badge
                    key={badge}
                    variant="secondary"
                    className="px-1.5 py-0 text-xs"
                  >
                    {badge}
                  </Badge>
                ))}
                {overflowCount > 0 && (
                  <Badge variant="outline" className="px-1.5 py-0 text-xs">
                    +{overflowCount}
                  </Badge>
                )}
              </>
            )}
          </div>
        </div>

        {showMapColumns && (
          <div className={cn(COL.city, 'hidden shrink-0 lg:block')}>
            {mapCityCode && (
              <span className="text-sm font-semibold text-foreground">
                {mapCityCode}
              </span>
            )}
          </div>
        )}

        {showMapColumns && (
          <div
            className={cn(
              COL.country,
              'hidden shrink-0 lg:flex items-center gap-1.5',
            )}
          >
            {CountryFlag && (
              <CountryFlag className="h-3 w-4 shrink-0 rounded-[1px]" />
            )}
            {mapCountry && (
              <span className="text-sm font-semibold text-foreground">
                {mapCountry}
              </span>
            )}
          </div>
        )}

        <div className={cn(COL.version, 'shrink-0')}>
          <span className="text-sm font-semibold text-foreground">
            {entry.installedVersion}
          </span>
        </div>

        <div
          className={cn(
            COL.actions,
            'shrink-0 flex items-center justify-end gap-0.5',
          )}
        >
          {pendingUpdate && (
            <Button
              variant="ghost"
              size="icon"
              className={cn('h-7 w-7', UPDATE_ICON_ACCENT)}
              onClick={() => setUpdateOpen(true)}
              aria-label="Update to latest"
              disabled={mutationLocked}
            >
              <CircleFadingArrowUp className="h-3.5 w-3.5" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            className={cn('h-7 w-7', FILES_ICON_ACCENT)}
            onClick={() => openInstallFolder(entry, metroMakerDataPath)}
            aria-label="Open install folder"
            disabled={!metroMakerDataPath}
          >
            <FolderOpen className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className={cn('h-7 w-7', UNINSTALL_ICON_ACCENT)}
            onClick={() => setUninstallOpen(true)}
            aria-label="Uninstall"
            disabled={mutationLocked}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </article>

      <AppDialog
        open={uninstallOpen}
        onOpenChange={setUninstallOpen}
        title="Uninstall"
        description="This will permanently remove all installed files. You can reinstall it later from the Browse page."
        icon={OctagonX}
        tone="uninstall"
        confirm={withLockAwareConfirm({
          label: 'Uninstall',
          onConfirm: handleUninstall,
          loading: uninstallLoading,
        }, mutationLocked, mutationLockedReason)}
      >
        <div className="rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
          <span className="font-medium text-foreground">{entry.item.name}</span>
        </div>
      </AppDialog>

      {updateOpen && updateTarget && (
        <AppDialog
          open={updateOpen}
          onOpenChange={setUpdateOpen}
          title={`Update`}
          description={`This will update the selected ${updateTarget.type === 'mod' ? 'mod' : 'map'} to its latest available version.`}
          icon={CircleFadingArrowUp}
          tone="update"
          confirm={withLockAwareConfirm({
            label: 'Update',
            onConfirm: handleUpdate,
            loading: updateLoading,
          }, mutationLocked, mutationLockedReason)}
        >
          {previewEntries.length > 0 && (
            <div className="max-h-48 overflow-y-auto rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
              <ul className="space-y-1">
                {previewEntries.map((e) => (
                  <li key={e.key} className="flex gap-2">
                    <span className="min-w-0 flex-1 truncate">{e.name}</span>
                    <span className="font-mono tabular-nums text-foreground">
                      {e.currentVersion} &rarr; {e.latestVersion}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </AppDialog>
      )}
    </>
  );
}
