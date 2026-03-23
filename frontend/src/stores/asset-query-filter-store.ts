import type { AssetType } from '@/lib/asset-types';
import type {
  AssetQueryFilterState,
  FilterByAssetType,
} from '@/stores/asset-type-filter-state';

export type AssetQueryFilters = AssetQueryFilterState;

export type AssetQueryFilterUpdater =
  | AssetQueryFilters
  | ((prev: AssetQueryFilters) => AssetQueryFilters);

export interface AssetQueryFilterStoreState {
  filters: AssetQueryFilters;
  page: number;
  scopedByType: FilterByAssetType;
  setFilters: (updater: AssetQueryFilterUpdater) => void;
  setType: (type: AssetType) => void;
  setPage: (page: number) => void;
}
