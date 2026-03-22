import {
  ChevronDown,
  ChevronUp,
  CircleFadingArrowUp,
  FolderOpen,
  Trash2,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { Link } from 'wouter';

import { UninstallDialog } from '@/components/dialogs/UninstallDialog';
import { UpdateSubscriptionsDialog } from '@/components/dialogs/UpdateSubscriptionsDialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { type InstalledTaggedItem } from '@/hooks/use-filtered-installed-items';
import type { AssetType } from '@/lib/asset-types';
import { assetTypeToListingPath } from '@/lib/asset-types';
import type { SortDirection, SortState } from '@/lib/constants';
import { toggleSortField } from '@/lib/constants';
import { getCountryFlagIcon } from '@/lib/flags';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { formatSourceQuality } from '@/lib/map-filter-values';
import { MAX_CARD_BADGES } from '@/lib/search';
import {
  composeAssetKey,
  getPendingSubscriptionUpdate,
  type PendingUpdatesByKey,
} from '@/lib/subscription-updates';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';
import { useLibraryStore } from '@/stores/library-store';

import { OpenInFileExplorer } from '../../../wailsjs/go/main/App';
import type { types } from '../../../wailsjs/go/models';

interface LibraryTableProps {
  items: InstalledTaggedItem[];
  activeType: AssetType;
  pendingUpdatesByKey: PendingUpdatesByKey;
  onRefreshPendingUpdates: () => Promise<void>;
  sort: SortState;
  onSortChange: (sort: SortState) => void;
}

interface SortableHeaderButtonProps {
  label: string;
  isActive: boolean;
  direction: SortDirection;
  onClick: () => void;
}

const UPDATE_ICON_ACCENT_CLASS = getLocalAccentClasses('update').iconButton;
const FILES_ICON_ACCENT_CLASS = getLocalAccentClasses('files').iconButton;
const UNINSTALL_ICON_ACCENT_CLASS =
  getLocalAccentClasses('uninstall').iconButton;

function SortableHeaderButton({
  label,
  isActive,
  direction,
  onClick,
}: SortableHeaderButtonProps) {
  const labelForAria = label.toLowerCase();
  const SortIcon = isActive && direction === 'asc' ? ChevronUp : ChevronDown;
  return (
    <Button
      type="button"
      onClick={onClick}
      intent="plain"
      size="xs"
      className="h-auto min-h-0 px-0 py-0 inline-flex items-center gap-1 text-foreground font-medium hover:bg-transparent hover:text-foreground/80 active:translate-y-0"
      aria-label={
        isActive && direction === 'asc'
          ? `Sort ${labelForAria} descending`
          : `Sort ${labelForAria} ascending`
      }
    >
      <span>{label}</span>
      <SortIcon
        className={cn('h-4 w-4', isActive ? 'opacity-100' : 'opacity-40')}
      />
    </Button>
  );
}

function composeItemKey(item: InstalledTaggedItem): string {
  return composeAssetKey(item.type, item.item.id);
}

function joinOsPath(...parts: string[]): string {
  return parts
    .filter(Boolean)
    .map((part, index) => {
      if (index === 0) return part.replace(/[\\/]+$/, '');
      return part.replace(/^[\\/]+|[\\/]+$/g, '');
    })
    .join('/');
}

export function LibraryTable({
  items,
  activeType,
  pendingUpdatesByKey,
  onRefreshPendingUpdates,
  sort,
  onSortChange,
}: LibraryTableProps) {
  const { selectedIds, toggleSelected, selectAll, clearSelection } =
    useLibraryStore();
  const showMapColumns = activeType === 'map';

  const allKeys = items.map(composeItemKey);
  const allSelected =
    items.length > 0 && allKeys.every((k) => selectedIds.has(k));
  const someSelected = !allSelected && allKeys.some((k) => selectedIds.has(k));

  const handleSelectAll = () => {
    if (allSelected) {
      clearSelection();
    } else {
      selectAll(allKeys);
    }
  };

  const isNameSorted = sort.field === 'name';
  const isCityCodeSorted = sort.field === 'city_code';
  const isCountrySorted = sort.field === 'country';

  return (
    <div className="rounded-md border border-border">
      <Table className="table-auto">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>
              <Checkbox
                checked={
                  allSelected ? true : someSelected ? 'indeterminate' : false
                }
                onCheckedChange={handleSelectAll}
                aria-label="Select all"
              />
            </TableHead>
            <TableHead>
              <SortableHeaderButton
                label="Name"
                isActive={isNameSorted}
                direction={sort.direction}
                onClick={() => onSortChange(toggleSortField(sort, 'name'))}
              />
            </TableHead>
            {showMapColumns && (
              <TableHead className="text-center">
                <SortableHeaderButton
                  label="City Code"
                  isActive={isCityCodeSorted}
                  direction={sort.direction}
                  onClick={() =>
                    onSortChange(toggleSortField(sort, 'city_code'))
                  }
                />
              </TableHead>
            )}
            <TableHead>Tags</TableHead>
            {showMapColumns && (
              <TableHead className="text-center">
                <SortableHeaderButton
                  label="Country"
                  isActive={isCountrySorted}
                  direction={sort.direction}
                  onClick={() => onSortChange(toggleSortField(sort, 'country'))}
                />
              </TableHead>
            )}
            <TableHead className="text-center">Version</TableHead>
            <TableHead></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((entry) => {
            const key = composeItemKey(entry);
            const isSelected = selectedIds.has(key);

            return (
              <LibraryTableRow
                key={key}
                entry={entry}
                pendingUpdatesByKey={pendingUpdatesByKey}
                isSelected={isSelected}
                showMapColumns={showMapColumns}
                onRefreshPendingUpdates={onRefreshPendingUpdates}
                onToggleSelect={() => toggleSelected(key)}
              />
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

interface LibraryTableRowProps {
  entry: InstalledTaggedItem;
  pendingUpdatesByKey: PendingUpdatesByKey;
  isSelected: boolean;
  showMapColumns: boolean;
  onRefreshPendingUpdates: () => Promise<void>;
  onToggleSelect: () => void;
}

function LibraryTableRow({
  entry,
  pendingUpdatesByKey,
  isSelected,
  showMapColumns,
  onRefreshPendingUpdates,
  onToggleSelect,
}: LibraryTableRowProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [updateOpen, setUpdateOpen] = useState(false);
  const actionButtonBaseClass = 'h-8 w-8 transition-colors';
  const removeSelected = useLibraryStore((s) => s.removeSelected);
  const metroMakerDataPath = useConfigStore(
    (s) => s.config?.metroMakerDataPath,
  );
  const isMap = entry.type === 'map';
  const map = isMap ? (entry.item as types.MapManifest) : null;
  const isLocalEntry = entry.isLocal;
  const isLocalMap = isMap && isLocalEntry;
  const mapBadges = map
    ? [
        map.location,
        formatSourceQuality(map.source_quality),
        map.level_of_detail,
        ...(map.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : [];
  const badges = isMap ? mapBadges : (entry.item.tags ?? []);
  const mapCityCode = map?.city_code?.trim().toUpperCase() ?? '';
  const mapCountry = map?.country?.trim().toUpperCase() ?? '';
  const CountryFlag = isMap ? getCountryFlagIcon(mapCountry) : null;
  const pendingUpdate = isLocalEntry
    ? undefined
    : getPendingSubscriptionUpdate(
        pendingUpdatesByKey,
        entry.type,
        entry.item.id,
      );

  const resolveInstallFolderPath = (): string | null => {
    if (!metroMakerDataPath) return null;

    if (entry.type === 'mod') {
      return joinOsPath(metroMakerDataPath, 'mods', entry.item.id);
    }

    const cityCode = (map?.city_code ?? '').trim();
    if (!cityCode) {
      return joinOsPath(metroMakerDataPath, 'cities', 'data');
    }

    return joinOsPath(metroMakerDataPath, 'cities', 'data', cityCode);
  };

  const handleOpenInstallFolder = () => {
    const folderPath = resolveInstallFolderPath();
    if (!folderPath) return;

    void (async () => {
      try {
        const result = await OpenInFileExplorer(folderPath);
        if (result?.status === 'error') {
          toast.error(result?.message || 'Failed to open install folder');
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : String(err));
      }
    })();
  };

  return (
    <>
      <TableRow
        data-state={isSelected ? 'selected' : undefined}
        className={cn('group transition-colors', isSelected && 'bg-muted/50')}
      >
        <TableCell>
          <Checkbox
            checked={isSelected}
            onCheckedChange={onToggleSelect}
            aria-label={`Select ${entry.item.name}`}
          />
        </TableCell>

        <TableCell>
          <div className="min-w-0">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 flex-wrap">
                {isLocalMap ? (
                  <span className="font-medium text-sm text-foreground truncate">
                    {entry.item.name}
                  </span>
                ) : (
                  <Link
                    href={`/project/${assetTypeToListingPath(entry.type)}/${entry.item.id}`}
                    className="font-medium text-sm text-foreground hover:underline truncate"
                  >
                    {entry.item.name}
                  </Link>
                )}
              </div>
              <p className="text-xs text-muted-foreground truncate">
                by {entry.item.author}
              </p>
            </div>
          </div>
        </TableCell>

        {showMapColumns && (
          <TableCell className="align-middle text-center">
            {isMap && mapCityCode ? (
              <span className="font-mono text-sm font-bold text-foreground">
                {mapCityCode}
              </span>
            ) : (
              <span className="block h-5" aria-hidden="true" />
            )}
          </TableCell>
        )}

        <TableCell className="align-middle">
          {isLocalEntry ? (
            <Badge
              variant="secondary"
              className="text-sm font-semibold uppercase tracking-wide px-2.5 py-0.5"
            >
              Local
            </Badge>
          ) : badges.length > 0 ? (
            <div
              className={cn(
                'flex flex-wrap items-center gap-1 self-center justify-start text-left',
                isMap && 'ml-1',
              )}
            >
              {badges.slice(0, MAX_CARD_BADGES).map((badge) => (
                <Badge
                  key={badge}
                  variant="secondary"
                  className="text-xs px-1.5 py-0"
                >
                  {badge}
                </Badge>
              ))}
              {badges.length > MAX_CARD_BADGES && (
                <Badge variant="outline" className="text-xs px-1.5 py-0">
                  +{badges.length - MAX_CARD_BADGES}
                </Badge>
              )}
            </div>
          ) : (
            <span className="block h-5" aria-hidden="true" />
          )}
        </TableCell>

        {showMapColumns && (
          <TableCell className="align-middle text-center">
            {isMap && mapCountry ? (
              <div className="mx-auto flex items-center justify-center gap-1.5 whitespace-nowrap">
                {CountryFlag && (
                  <CountryFlag className="h-3.5 w-5 rounded-[1px] shrink-0" />
                )}
                <span className="font-mono text-sm font-bold text-foreground">
                  {mapCountry}
                </span>
              </div>
            ) : (
              <span className="block h-5" aria-hidden="true" />
            )}
          </TableCell>
        )}

        <TableCell className="align-middle text-center">
          <p className="text-sm font-mono tabular-nums text-foreground text-center whitespace-nowrap">
            {entry.installedVersion}
          </p>
        </TableCell>

        <TableCell>
          <div className="flex items-center justify-end gap-1">
            {pendingUpdate && (
              <Button
                variant="ghost"
                size="icon"
                className={cn(actionButtonBaseClass, UPDATE_ICON_ACCENT_CLASS)}
                onClick={() => setUpdateOpen(true)}
                aria-label="Update to latest"
              >
                <CircleFadingArrowUp className="h-4 w-4" />
              </Button>
            )}

            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <Button
                variant="ghost"
                size="icon"
                className={cn(actionButtonBaseClass, FILES_ICON_ACCENT_CLASS)}
                onClick={handleOpenInstallFolder}
                aria-label="Open install folder"
                disabled={!metroMakerDataPath}
              >
                <FolderOpen className="h-4 w-4" />
              </Button>

              <Button
                variant="ghost"
                size="icon"
                className={cn(
                  actionButtonBaseClass,
                  UNINSTALL_ICON_ACCENT_CLASS,
                )}
                onClick={() => setUninstallOpen(true)}
                aria-label="Delete"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </TableCell>
      </TableRow>

      {uninstallOpen && (
        <UninstallDialog
          open={uninstallOpen}
          onOpenChange={setUninstallOpen}
          onUninstallSuccess={() => {
            removeSelected([composeAssetKey(entry.type, entry.item.id)]);
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
          onUpdateSuccess={() => {
            void onRefreshPendingUpdates();
          }}
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
