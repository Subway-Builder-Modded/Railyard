import type { SortDirection, SortState } from '@/lib/constants';

import type { types } from '../../wailsjs/go/models';

export type TaggedItem =
  | { type: 'mod'; item: types.ModManifest }
  | { type: 'map'; item: types.MapManifest };

export function buildTaggedItems(
  mods: types.ModManifest[],
  maps: types.MapManifest[],
): TaggedItem[] {
  const modItems: TaggedItem[] = mods.map((item) => ({ type: 'mod', item }));
  const mapItems: TaggedItem[] = maps.map((item) => ({ type: 'map', item }));
  return [...modItems, ...mapItems];
}

export function compareByDirection(
  a: number,
  b: number,
  direction: SortDirection,
): number {
  return direction === 'asc' ? a - b : b - a;
}

export function getTotalDownloads(
  item: TaggedItem,
  modDownloadTotals: Record<string, number>,
  mapDownloadTotals: Record<string, number>,
): number {
  return item.type === 'mod'
    ? (modDownloadTotals[item.item.id] ?? 0)
    : (mapDownloadTotals[item.item.id] ?? 0);
}

export function getLastUpdated(item: TaggedItem): number {
  const timestamp = item.item.last_updated;
  return typeof timestamp === 'number' && Number.isFinite(timestamp)
    ? timestamp
    : 0;
}

export function compareItems(
  a: TaggedItem,
  b: TaggedItem,
  sort: SortState,
  modDownloadTotals: Record<string, number>,
  mapDownloadTotals: Record<string, number>,
): number {
  const compareText = (
    left: string,
    right: string,
    direction: SortDirection,
  ) =>
    direction === 'asc' ? left.localeCompare(right) : right.localeCompare(left);

  switch (sort.field) {
    case 'name':
      return compareText(a.item.name ?? '', b.item.name ?? '', sort.direction);
    case 'country': {
      const countryA =
        a.type === 'map' ? ((a.item as types.MapManifest).country ?? '') : '';
      const countryB =
        b.type === 'map' ? ((b.item as types.MapManifest).country ?? '') : '';
      return compareText(countryA, countryB, sort.direction);
    }
    case 'author':
      return compareText(
        a.item.author ?? '',
        b.item.author ?? '',
        sort.direction,
      );
    case 'population': {
      const popA =
        a.type === 'map' ? ((a.item as types.MapManifest).population ?? 0) : 0;
      const popB =
        b.type === 'map' ? ((b.item as types.MapManifest).population ?? 0) : 0;
      return compareByDirection(popA, popB, sort.direction);
    }
    case 'downloads': {
      const downloadsA = getTotalDownloads(
        a,
        modDownloadTotals,
        mapDownloadTotals,
      );
      const downloadsB = getTotalDownloads(
        b,
        modDownloadTotals,
        mapDownloadTotals,
      );
      return compareByDirection(downloadsA, downloadsB, sort.direction);
    }
    case 'last_updated': {
      const updatedA = getLastUpdated(a);
      const updatedB = getLastUpdated(b);
      return compareByDirection(updatedA, updatedB, sort.direction);
    }
    default:
      return 0;
  }
}

export function sortTaggedItemsByLastUpdated<T extends TaggedItem>(
  items: T[],
  direction: SortDirection = 'desc',
): T[] {
  return [...items].sort((a, b) =>
    compareItems(
      a,
      b,
      { field: 'last_updated', direction },
      {},
      {},
    ),
  );
}