export const PER_PAGE_OPTIONS = [12, 24, 48] as const;
export type PerPage = typeof PER_PAGE_OPTIONS[number];

export type ListingType = "mods" | "maps";

// Union type of valid sort dimensions (TODO: Include "updated" once the corresponding BE endpoint is added)
export type SortField = "name" | "author" | "population" | "downloads" | "random";
// Union type of valid sort directions
export type SortDirection = "asc" | "desc";
export type SortKey = `${SortField}:${SortDirection}`; // Template literal type to help encapsulate logic of converting between SortState and string keys

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

const SORT_FIELDS = ["name", "author", "population", "downloads", "random"] as const;
const SORT_DIRECTIONS = ["asc", "desc"] as const;

function directionsForField(field: SortField): readonly SortDirection[] {
  return field === "random" ? (["asc"] as const) : SORT_DIRECTIONS;
}

function sortOptionLabel(field: SortField, direction: SortDirection): string {
  switch (field) {
    case "name":
      return direction === "asc" ? "Name A-Z" : "Name Z-A";
    case "author":
      return direction === "asc" ? "Author A-Z" : "Author Z-A";
    case "population":
      return direction === "asc" ? "Population Low-High" : "Population High-Low";
    case "downloads":
      return direction === "asc" ? "Total Downloads \u2191" : "Total Downloads \u2193";
    case "random":
      return "Random";
    default: // Default case to ensure all fields are handled. Programmer error if this is ever reached
      throw new Error(`Unhandled sort field: ${String(field)}`);
  }
}

export const SORT_OPTIONS = SORT_FIELDS.flatMap((field) =>
  directionsForField(field).map((direction) => ({
    value: `${field}:${direction}` as SortKey,
    label: sortOptionLabel(field, direction),
    sort: { field, direction },
    mapOnly: field === "population",
  }))
) satisfies SortOption[];

export const DEFAULT_SORT_STATE: SortState = { field: "name", direction: "asc" };

export function getSortOptionsForType(type: ListingType): SortOption[] {
  return SORT_OPTIONS.filter((opt) => !opt.mapOnly || type === "maps");
}

const SORT_STATE_BY_KEY = Object.fromEntries(
  SORT_OPTIONS.map((option) => [option.value, option.sort])
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

export function sortStateToOptionKey(state: SortState, type: ListingType): SortKey {
  const options = getSortOptionsForType(type);
  const requestedKey = SortKey.fromState(state);
  const option =
    options.find((opt) => SortKey.equals(opt.value, requestedKey)) ??
    options.find((opt) => opt.sort.field === state.field) ??
    options[0] ??
    SORT_OPTIONS[0];

  return option.value;
}
