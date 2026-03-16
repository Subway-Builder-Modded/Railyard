import { ASSET_TYPES, type AssetType } from '@/lib/asset-types';
import {
  DEFAULT_SORT_STATE,
  type PerPage,
  type SortState,
} from '@/lib/constants';

export interface AssetFilterFields {
  sort: SortState;
  randomSeed: number;
  mod: {
    tags: string[];
  };
  map: {
    locations: string[];
    sourceQuality: string[];
    levelOfDetail: string[];
    specialDemand: string[];
  };
}

export interface AssetQueryFilterState extends AssetFilterFields {
  query: string;
  type: AssetType;
  perPage: PerPage;
}

export interface AssetFilterState extends AssetFilterFields {
  page: number;
}

export type FilterByAssetType = Record<AssetType, AssetFilterState>;

export function createRandomSeed(): number {
  return Math.floor(Math.random() * 2_147_483_647);
}

export const defaultSearchFilters: AssetQueryFilterState = {
  query: '',
  type: 'map',
  sort: DEFAULT_SORT_STATE,
  randomSeed: createRandomSeed(),
  perPage: 12,
  mod: {
    tags: [],
  },
  map: {
    locations: [],
    sourceQuality: [],
    levelOfDetail: [],
    specialDemand: [],
  },
};

export const defaultLibraryFilters: AssetQueryFilterState = {
  ...defaultSearchFilters,
  sort: {
    ...DEFAULT_SORT_STATE,
    field: 'name',
    direction: 'asc',
  },
};

function cloneFilterFields(fields: AssetFilterFields): AssetFilterFields {
  return {
    sort: { ...fields.sort },
    randomSeed: fields.randomSeed,
    mod: {
      tags: [...fields.mod.tags],
    },
    map: {
      locations: [...fields.map.locations],
      sourceQuality: [...fields.map.sourceQuality],
      levelOfDetail: [...fields.map.levelOfDetail],
      specialDemand: [...fields.map.specialDemand],
    },
  };
}

export function cloneFilterState<T extends AssetQueryFilterState>(state: T): T {
  return {
    ...state,
    ...cloneFilterFields(state),
  };
}

export function toAssetFilterState(
  fields: AssetFilterFields,
  page: number,
): AssetFilterState {
  return {
    ...cloneFilterFields(fields),
    page,
  };
}

// Initialize a filter state for each asset type based on the provided filter values and page number
export function createFilterByAssetType(
  fields: AssetFilterFields,
  page: number,
): FilterByAssetType {
  return Object.fromEntries(
    ASSET_TYPES.map((assetType) => [
      assetType,
      toAssetFilterState(fields, page),
    ]),
  ) as FilterByAssetType;
}

export function syncFilter<T extends AssetFilterFields & { type: AssetType }>(
  scopedByType: FilterByAssetType,
  filters: T,
  page: number,
): FilterByAssetType {
  return {
    ...scopedByType,
    [filters.type]: toAssetFilterState(filters, page),
  };
}

export function applyFilter<T extends AssetFilterFields & { type: AssetType }>(
  filters: T,
  nextType: AssetType,
  scopedState: AssetFilterState,
): T {
  return {
    ...filters,
    ...cloneFilterFields(scopedState),
    type: nextType,
  };
}

// Switch the filter to a different asset type, syncing the current filter values to the new type's state
export function switchFilter<T extends AssetFilterFields & { type: AssetType }>(
  filters: T,
  page: number,
  scopedByType: FilterByAssetType,
  nextType: AssetType,
): {
  filters: T;
  page: number;
  scopedByType: FilterByAssetType;
} {
  const nextScopedByType = syncFilter(scopedByType, filters, page);
  const targetState = nextScopedByType[nextType];

  return {
    filters: applyFilter(filters, nextType, targetState),
    page: targetState.page,
    scopedByType: nextScopedByType,
  };
}
