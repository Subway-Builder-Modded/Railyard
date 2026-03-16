export const SEARCH_VIEW_MODES = ['full', 'compact', 'list'] as const;

export type SearchViewMode = (typeof SEARCH_VIEW_MODES)[number];

export function isSearchViewMode(value: unknown): value is SearchViewMode {
  return (
    typeof value === 'string' &&
    (SEARCH_VIEW_MODES as readonly string[]).includes(value)
  );
}

export function normalizeSearchViewMode(
  value: unknown,
  fallback: SearchViewMode = 'full',
): SearchViewMode {
  return isSearchViewMode(value) ? value : fallback;
}
