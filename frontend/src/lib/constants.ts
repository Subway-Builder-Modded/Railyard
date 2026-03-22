import type { AssetType } from '@/lib/asset-types';

export const PER_PAGE_OPTIONS = [12, 24, 48] as const;
export type PerPage = (typeof PER_PAGE_OPTIONS)[number];

export type SortField =
  | 'name'
  | 'city_code'
  | 'country'
  | 'author'
  | 'population'
  | 'downloads'
  | 'last_updated'
  | 'random';
// Union type of valid sort directions
export type SortDirection = 'asc' | 'desc';
export type SortKey = `${SortField}:${SortDirection}`;

export interface SortState {
  field: SortField;
  direction: SortDirection;
}

export interface SortOption {
  value: SortKey;
  label: string;
  sort: SortState;
  mapOnly?: boolean;
}

const SORT_FIELDS = [
  'last_updated',
  'downloads',
  'population',
  'name',
  'city_code',
  'country',
  'author',
  'random',
] as const;
const DESC_ASC_DIRECTIONS = ['desc', 'asc'] as const;

function directionsForField(field: SortField): readonly SortDirection[] {
  if (field === 'random') return ['asc'] as const;
  if (
    field === 'name' ||
    field === 'city_code' ||
    field === 'country' ||
    field === 'author'
  ) {
    return ['asc', 'desc'] as const;
  }
  return DESC_ASC_DIRECTIONS;
}

function sortOptionLabel(field: SortField): string {
  switch (field) {
    case 'name':
      return 'Name';
    case 'city_code':
      return 'City Code';
    case 'country':
      return 'Country';
    case 'author':
      return 'Author';
    case 'population':
      return 'Population';
    case 'downloads':
      return 'Downloads';
    case 'last_updated':
      return 'Last Updated';
    case 'random':
      return 'Random';
    default: // Default case to ensure all fields are handled. Programmer error if this is ever reached
      throw new Error(`Unhandled sort field: ${String(field)}`);
  }
}

export const SORT_OPTIONS = SORT_FIELDS.flatMap((field) =>
  directionsForField(field).map((direction) => ({
    value: `${field}:${direction}` as SortKey,
    label: sortOptionLabel(field),
    sort: { field, direction },
    mapOnly:
      field === 'population' || field === 'city_code' || field === 'country',
  })),
) satisfies SortOption[];

export const DEFAULT_SORT_STATE: SortState = {
  field: 'last_updated',
  direction: 'desc',
};

export function getSortOptionsForType(type: AssetType): SortOption[] {
  return SORT_OPTIONS.filter((opt) => !opt.mapOnly || type === 'map');
}

const SORT_STATE_BY_KEY = Object.fromEntries(
  SORT_OPTIONS.map((option) => [option.value, option.sort]),
) as Record<SortKey, SortState>;

export const SortKey = {
  equals(left: SortKey, right: SortKey): boolean {
    return left === right;
  },
  fromState(state: SortState): SortKey {
    return `${state.field}:${state.direction}`;
  },
  toState(value: string): SortState | undefined {
    return SORT_STATE_BY_KEY[value as SortKey];
  },
} as const;

export function sortKeyToState(value: string): SortState {
  return SortKey.toState(value) ?? DEFAULT_SORT_STATE;
}

export function sortStateToOptionKey(
  state: SortState,
  type: AssetType,
): SortKey {
  const options = getSortOptionsForType(type);
  const requestedKey = SortKey.fromState(state);
  const defaultKey = SortKey.fromState(DEFAULT_SORT_STATE);
  const defaultOption =
    options.find((opt) => SortKey.equals(opt.value, defaultKey)) ??
    options[0] ??
    SORT_OPTIONS[0];
  const option =
    options.find((opt) => SortKey.equals(opt.value, requestedKey)) ??
    options.find((opt) => opt.sort.field === state.field) ??
    defaultOption;

  return option.value;
}

export function toggleSortField(
  current: SortState,
  field: Exclude<SortField, 'random'>,
): SortState {
  if (current.field === field) {
    return {
      field,
      direction: current.direction === 'asc' ? 'desc' : 'asc',
    };
  }

  return {
    field,
    direction: 'asc',
  };
}

export function normalizeSortStateForType(
  state: SortState,
  type: AssetType,
): SortState {
  const options = getSortOptionsForType(type);
  const requestedKey = SortKey.fromState(state);
  const requested = options.find((option) =>
    SortKey.equals(option.value, requestedKey),
  );
  if (requested) {
    return requested.sort;
  }

  const fallbackField = type === 'map' ? 'name' : 'name';
  const fallback = options.find(
    (option) => option.sort.field === fallbackField,
  );
  return fallback?.sort ?? options[0]?.sort ?? DEFAULT_SORT_STATE;
}
