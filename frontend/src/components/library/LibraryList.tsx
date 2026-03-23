import {
  ArrowDown,
  ArrowUp,
  CircleFadingArrowUp,
  FolderOpen,
  Globe,
  Hash,
  HardDrive,
  Trash2,
  Type,
} from 'lucide-react';
import { type ComponentType, useCallback, useState } from 'react';
import { Link } from 'wouter';

import { UninstallDialog } from '@/components/dialogs/UninstallDialog';
import { UpdateSubscriptionsDialog } from '@/components/dialogs/UpdateSubscriptionsDialog';
import { GalleryImage } from '@/components/shared/GalleryImage';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import type { InstalledTaggedItem } from '@/hooks/use-filtered-installed-items';
import type { AssetType } from '@/lib/asset-types';
import { assetTypeToListingPath } from '@/lib/asset-types';
import { type SortDirection, type SortField, type SortState, TEXT_SORT_FIELDS } from '@/lib/constants';
import { getCountryFlagIcon } from '@/lib/flags';
import { openInstallFolder } from '@/lib/install-path';
import { LOCAL_ACCENTS } from '@/lib/local-accent';
import { formatSourceQuality } from '@/lib/map-filter-values';
import {
  composeAssetKey,
  getPendingSubscriptionUpdate,
  type PendingUpdatesByKey,
} from '@/lib/subscription-updates';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';
import { useLibraryStore } from '@/stores/library-store';

import type { types } from '../../../wailsjs/go/models';

const UPDATE_ICON_ACCENT = LOCAL_ACCENTS.update.iconButton;
const FILES_ICON_ACCENT = LOCAL_ACCENTS.files.iconButton;
const UNINSTALL_ICON_ACCENT = LOCAL_ACCENTS.uninstall.iconButton;

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

interface SortableHeaderCellProps {
  label: string;
  field: Exclude<SortField, 'random'>;
  icon: ComponentType<{ className?: string }>;
  sort: SortState;
  onSort: (field: Exclude<SortField, 'random'>) => void;
  className?: string;
}

function SortableHeaderCell({ label, field, icon: Icon, sort, onSort, className }: SortableHeaderCellProps) {
  const isActive = sort.field === field;
  const invert = TEXT_SORT_FIELDS.has(field);
  const showUp = isActive && (invert ? sort.direction === 'desc' : sort.direction === 'asc');
  const SortIcon = showUp ? ArrowUp : ArrowDown;

  return (
    <button
      type="button"
      onClick={() => onSort(field)}
      className={cn(
        'inline-flex items-center gap-1 text-xs font-semibold uppercase tracking-wide transition-colors',
        isActive ? 'text-foreground' : 'text-muted-foreground hover:text-foreground',
        className,
      )}
      aria-label={`Sort by ${label} ${isActive && sort.direction === 'asc' ? 'descending' : 'ascending'}`}
    >
      <Icon className="h-3.5 w-3.5 shrink-0" />
      {label}
      <SortIcon
        className={cn('h-3.5 w-3.5 shrink-0', isActive ? 'opacity-100' : 'opacity-30')}
      />
    </button>
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
  const allSelected = items.length > 0 && allKeys.every((k) => selectedIds.has(k));
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
          onCheckedChange={() => (allSelected ? clearSelection() : selectAll(allKeys))}
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
                onSort={handleColumnSort}
              />
            </div>
            <div className={cn(COL.country, 'hidden shrink-0 lg:block')}>
              <SortableHeaderCell
                label="Country"
                field="country"
                icon={Globe}
                sort={sort}
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
}

function LibraryListRow({
  entry,
  showMapColumns,
  pendingUpdatesByKey,
  onRefreshPendingUpdates,
}: LibraryListRowProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [updateOpen, setUpdateOpen] = useState(false);

  const { selectedIds, toggleSelected, removeSelected } = useLibraryStore();
  const metroMakerDataPath = useConfigStore((s) => s.config?.metroMakerDataPath);

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
    : getPendingSubscriptionUpdate(pendingUpdatesByKey, entry.type, entry.item.id);

  const projectHref = `/project/${assetTypeToListingPath(entry.type)}/${entry.item.id}`;

  const visibleBadges = badges.slice(0, 2);
  const overflowCount = badges.length - visibleBadges.length;

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
                  <Badge key={badge} variant="secondary" className="px-1.5 py-0 text-xs">
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
              <span className="text-sm font-semibold text-foreground">{mapCityCode}</span>
            )}
          </div>
        )}

        {showMapColumns && (
          <div className={cn(COL.country, 'hidden shrink-0 lg:flex items-center gap-1.5')}>
            {CountryFlag && <CountryFlag className="h-3 w-4 shrink-0 rounded-[1px]" />}
            {mapCountry && (
              <span className="text-sm font-semibold text-foreground">{mapCountry}</span>
            )}
          </div>
        )}

        <div className={cn(COL.version, 'shrink-0')}>
          <span className="text-sm font-semibold text-foreground">
            {entry.installedVersion}
          </span>
        </div>

        <div className={cn(COL.actions, 'shrink-0 flex items-center justify-end gap-0.5')}>
          {pendingUpdate && (
            <Button
              variant="ghost"
              size="icon"
              className={cn('h-7 w-7', UPDATE_ICON_ACCENT)}
              onClick={() => setUpdateOpen(true)}
              aria-label="Update to latest"
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
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </article>

      {uninstallOpen && (
        <UninstallDialog
          open={uninstallOpen}
          onOpenChange={setUninstallOpen}
          onUninstallSuccess={() => {
            removeSelected([key]);
            void onRefreshPendingUpdates();
          }}
          type={entry.type}
          id={entry.item.id}
          name={entry.item.name}
        />
      )}

      {updateOpen && pendingUpdate && (
        <UpdateSubscriptionsDialog
          open={updateOpen}
          onOpenChange={setUpdateOpen}
          onUpdateSuccess={() => void onRefreshPendingUpdates()}
          targets={[
            {
              id: entry.item.id,
              type: entry.type,
              name: entry.item.name,
              currentVersion: pendingUpdate.currentVersion,
              latestVersion: pendingUpdate.latestVersion,
            },
          ]}
        />
      )}
    </>
  );
}
