import type { PerPage } from "@/lib/constants";
import type { AssetType } from "@/lib/asset-types";

export type AssetFilterType = AssetType;

export interface SharedAssetFilterState {
  query: string;
  type: AssetFilterType;
  perPage: PerPage;
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

export function createDefaultSharedAssetFilters(
  type: AssetFilterType,
  perPage: PerPage = 12,
): SharedAssetFilterState {
  return {
    query: "",
    type,
    perPage,
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
}
