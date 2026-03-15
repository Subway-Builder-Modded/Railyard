import { ASSET_TYPES, type AssetType } from '@/lib/asset-types';
import type { SortState } from '@/lib/constants';

export interface TypeScopedFilterFields {
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

export interface TypeScopedFilterState extends TypeScopedFilterFields {
  page: number;
}

export type TypeScopedByAssetType = Record<AssetType, TypeScopedFilterState>;

function cloneFilterFields(fields: TypeScopedFilterFields): TypeScopedFilterFields {
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

export function toTypeScopedState(
  fields: TypeScopedFilterFields,
  page: number,
): TypeScopedFilterState {
  return {
    ...cloneFilterFields(fields),
    page,
  };
}

export function createTypeScopedByAssetType(
  fields: TypeScopedFilterFields,
  page: number,
): TypeScopedByAssetType {
  return Object.fromEntries(
    ASSET_TYPES.map((assetType) => [assetType, toTypeScopedState(fields, page)]),
  ) as TypeScopedByAssetType;
}

export function syncCurrentTypeScopedState<T extends TypeScopedFilterFields & { type: AssetType }>(
  scopedByType: TypeScopedByAssetType,
  filters: T,
  page: number,
): TypeScopedByAssetType {
  return {
    ...scopedByType,
    [filters.type]: toTypeScopedState(filters, page),
  };
}

export function applyTypeScopedState<T extends TypeScopedFilterFields & { type: AssetType }>(
  filters: T,
  nextType: AssetType,
  scopedState: TypeScopedFilterState,
): T {
  return {
    ...filters,
    ...cloneFilterFields(scopedState),
    type: nextType,
  };
}

export function switchTypeScopedState<T extends TypeScopedFilterFields & { type: AssetType }>(
  filters: T,
  page: number,
  scopedByType: TypeScopedByAssetType,
  nextType: AssetType,
): {
  filters: T;
  page: number;
  scopedByType: TypeScopedByAssetType;
} {
  const nextScopedByType = syncCurrentTypeScopedState(scopedByType, filters, page);
  const targetState = nextScopedByType[nextType];

  return {
    filters: applyTypeScopedState(filters, nextType, targetState),
    page: targetState.page,
    scopedByType: nextScopedByType,
  };
}
