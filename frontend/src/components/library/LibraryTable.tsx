import { ChevronDown, ChevronUp, FolderOpen, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { Link } from 'wouter';

import { UninstallDialog } from '@/components/dialogs/UninstallDialog';
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
import type { SortState } from '@/lib/constants';
import { toggleSortField } from '@/lib/constants';
import { getCountryFlagIcon } from '@/lib/flags';
import { formatSourceQuality } from '@/lib/map-filter-values';
import { MAX_CARD_BADGES } from '@/lib/search';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';
import { useLibraryStore } from '@/stores/library-store';

import { OpenInFileExplorer } from '../../../wailsjs/go/main/App';
import type { types } from '../../../wailsjs/go/models';

interface LibraryTableProps {
  items: InstalledTaggedItem[];
  activeType: AssetType;
  sort: SortState;
  onSortChange: (sort: SortState) => void;
}

function composeItemKey(item: InstalledTaggedItem): string {
  return `${item.type}-${item.item.id}`;
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
  sort,
  onSortChange,
}: LibraryTableProps) {
  const { selectedIds, toggleSelected, selectAll, clearSelection } =
    useLibraryStore();
  const showCountryColumn = activeType === 'map';

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
  const NameSortIcon =
    isNameSorted && sort.direction === 'asc' ? ChevronUp : ChevronDown;
  const isCountrySorted = sort.field === 'country';
  const CountrySortIcon =
    isCountrySorted && sort.direction === 'asc' ? ChevronUp : ChevronDown;

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="w-10">
              <Checkbox
                checked={
                  allSelected ? true : someSelected ? 'indeterminate' : false
                }
                onCheckedChange={handleSelectAll}
                aria-label="Select all"
              />
            </TableHead>
            <TableHead>
              <button
                type="button"
                onClick={() => onSortChange(toggleSortField(sort, 'name'))}
                className="inline-flex items-center gap-1 text-foreground font-medium hover:text-foreground/80 transition-colors"
                aria-label={
                  isNameSorted && sort.direction === 'asc'
                    ? 'Sort name descending'
                    : 'Sort name ascending'
                }
              >
                <span>Name</span>
                <NameSortIcon
                  className={cn(
                    'h-4 w-4',
                    isNameSorted ? 'opacity-100' : 'opacity-40',
                  )}
                />
              </button>
            </TableHead>
            {showCountryColumn && (
              <TableHead className="w-32 text-center">
                <button
                  type="button"
                  onClick={() => onSortChange(toggleSortField(sort, 'country'))}
                  className="inline-flex items-center gap-1 text-foreground font-medium hover:text-foreground/80 transition-colors"
                  aria-label={
                    isCountrySorted && sort.direction === 'asc'
                      ? 'Sort country descending'
                      : 'Sort country ascending'
                  }
                >
                  <span>Country</span>
                  <CountrySortIcon
                    className={cn(
                      'h-4 w-4',
                      isCountrySorted ? 'opacity-100' : 'opacity-40',
                    )}
                  />
                </button>
              </TableHead>
            )}
            <TableHead className="w-28 text-center">Version</TableHead>
            <TableHead className="w-24"></TableHead>
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
                isSelected={isSelected}
                showCountryColumn={showCountryColumn}
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
  isSelected: boolean;
  showCountryColumn: boolean;
  onToggleSelect: () => void;
}

function LibraryTableRow({
  entry,
  isSelected,
  showCountryColumn,
  onToggleSelect,
}: LibraryTableRowProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const removeSelected = useLibraryStore((s) => s.removeSelected);
  const metroMakerDataPath = useConfigStore(
    (s) => s.config?.metroMakerDataPath,
  );
  const isMap = entry.type === 'map';
  const map = isMap ? (entry.item as types.MapManifest) : null;
  const mapBadges = map
    ? [
        map.location,
        formatSourceQuality(map.source_quality),
        map.level_of_detail,
        ...(map.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : [];
  const badges = isMap ? mapBadges : (entry.item.tags ?? []);
  const mapCountry = map?.country?.trim().toUpperCase() ?? '';
  const CountryFlag = isMap ? getCountryFlagIcon(mapCountry) : null;

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
            <div className="flex items-center justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <Link
                    href={`/project/${assetTypeToListingPath(entry.type)}/${entry.item.id}`}
                    className="font-medium text-sm text-foreground hover:underline truncate"
                  >
                    {entry.item.name}
                  </Link>
                </div>
                <p className="text-xs text-muted-foreground truncate">
                  by {entry.item.author}
                </p>
              </div>

              {badges.length > 0 && (
                <div
                  className={cn(
                    'shrink-0 flex items-center gap-1 self-center justify-start text-left',
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
              )}
            </div>
          </div>
        </TableCell>

        {showCountryColumn && (
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
          <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-yellow-500 hover:text-yellow-400 hover:bg-yellow-500/10"
              onClick={handleOpenInstallFolder}
              aria-label="Open install folder"
              disabled={!metroMakerDataPath}
            >
              <FolderOpen className="h-4 w-4" />
            </Button>

            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setUninstallOpen(true)}
              aria-label="Delete"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </TableCell>
      </TableRow>

      {uninstallOpen && (
        <UninstallDialog
          open={uninstallOpen}
          onOpenChange={setUninstallOpen}
          onUninstallSuccess={() => {
            removeSelected([`${entry.type}-${entry.item.id}`]);
          }}
          type={entry.type}
          id={entry.item.id}
          name={entry.item.name}
        />
      )}
    </>
  );
}
