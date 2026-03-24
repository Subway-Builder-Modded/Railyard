import { describe, expect, it } from 'vitest';

import {
  applyFilter,
  cloneFilterState,
  createFilterByAssetType,
  createRandomSeed,
  defaultLibraryFilters,
  defaultSearchFilters,
  switchFilter,
  syncFilter,
  toAssetFilterState,
} from './asset-type-filter-state';

describe('createRandomSeed', () => {
  it('returns a positive integer within range', () => {
    const seed = createRandomSeed();
    expect(seed).toBeGreaterThanOrEqual(0);
    expect(seed).toBeLessThan(2_147_483_647);
    expect(Number.isInteger(seed)).toBe(true);
  });

  it('produces different values on successive calls', () => {
    const seeds = new Set(Array.from({ length: 20 }, () => createRandomSeed()));
    expect(seeds.size).toBeGreaterThan(1);
  });
});

describe('cloneFilterState', () => {
  it('creates a deep clone of filter fields', () => {
    const original = {
      ...defaultSearchFilters,
      mod: { tags: ['ui'] },
      map: {
        locations: ['europe'],
        sourceQuality: ['high-quality'],
        levelOfDetail: [],
        specialDemand: [],
      },
    };

    const cloned = cloneFilterState(original);

    // Verify equal values
    expect(cloned.mod.tags).toEqual(['ui']);
    expect(cloned.map.locations).toEqual(['europe']);

    // Verify independence
    cloned.mod.tags.push('gameplay');
    cloned.map.locations.push('asia');
    expect(original.mod.tags).toEqual(['ui']);
    expect(original.map.locations).toEqual(['europe']);
  });
});

describe('toAssetFilterState', () => {
  it('copies filter fields and attaches page', () => {
    const state = toAssetFilterState(
      { ...defaultSearchFilters, mod: { tags: ['ui'] } },
      3,
    );
    expect(state.page).toBe(3);
    expect(state.mod.tags).toEqual(['ui']);
  });

  it('clones arrays so mutations are isolated', () => {
    const source = { ...defaultSearchFilters, mod: { tags: ['ui'] } };
    const state = toAssetFilterState(source, 1);
    state.mod.tags.push('extra');
    expect(source.mod.tags).toEqual(['ui']);
  });
});

describe('createFilterByAssetType', () => {
  it('creates an entry for each asset type', () => {
    const byType = createFilterByAssetType(defaultSearchFilters, 2);
    expect(byType).toHaveProperty('mod');
    expect(byType).toHaveProperty('map');
    expect(byType.mod.page).toBe(2);
    expect(byType.map.page).toBe(2);
  });

  it('each entry is an independent clone', () => {
    const byType = createFilterByAssetType(defaultSearchFilters, 1);
    byType.mod.mod.tags.push('mutated');
    expect(byType.map.mod.tags).toEqual([]);
  });
});

describe('syncFilter', () => {
  it('updates the scoped state for the active type only', () => {
    const initial = createFilterByAssetType(defaultSearchFilters, 1);
    const updated = syncFilter(
      initial,
      { ...defaultSearchFilters, type: 'mod', mod: { tags: ['ui'] } },
      5,
    );

    expect(updated.mod.mod.tags).toEqual(['ui']);
    expect(updated.mod.page).toBe(5);
    expect(updated.map.mod.tags).toEqual([]); // untouched
    expect(updated.map.page).toBe(1); // untouched
  });
});

describe('applyFilter', () => {
  it('overlays scoped state onto the base filters with the new type', () => {
    const base = {
      ...defaultSearchFilters,
      type: 'mod' as const,
      query: 'test',
      perPage: 24 as const,
    };
    const scopedState = toAssetFilterState(
      { ...defaultSearchFilters, mod: { tags: ['gameplay'] } },
      4,
    );

    const result = applyFilter(base, 'map', scopedState);

    expect(result.type).toBe('map');
    expect(result.query).toBe('test'); // preserved from base
    expect(result.perPage).toBe(24); // preserved from base
    expect(result.mod.tags).toEqual(['gameplay']); // from scoped state
  });
});

describe('switchFilter', () => {
  it('saves the current type state and restores the target type state', () => {
    const initial = createFilterByAssetType(defaultSearchFilters, 1);

    // Simulate having changed mod state
    const modFilters = {
      ...defaultSearchFilters,
      type: 'mod' as const,
      mod: { tags: ['ui'] },
    };
    const step1 = switchFilter(modFilters, 3, initial, 'map');

    // mod state should be saved with page 3
    expect(step1.scopedByType.mod.mod.tags).toEqual(['ui']);
    expect(step1.scopedByType.mod.page).toBe(3);

    // active filters should be map type
    expect(step1.filters.type).toBe('map');
    expect(step1.page).toBe(1); // map page was 1
  });

  it('round-trips type switches preserving each type independently', () => {
    const initial = createFilterByAssetType(defaultSearchFilters, 1);

    // Set up map state
    const mapFilters = {
      ...defaultSearchFilters,
      type: 'map' as const,
      map: {
        locations: ['europe'],
        sourceQuality: [],
        levelOfDetail: [],
        specialDemand: [],
      },
    };
    const afterMapEdit = switchFilter(mapFilters, 2, initial, 'mod');
    expect(afterMapEdit.filters.type).toBe('mod');

    // Now switch back to map
    const afterSwitchBack = switchFilter(
      afterMapEdit.filters,
      afterMapEdit.page,
      afterMapEdit.scopedByType,
      'map',
    );
    expect(afterSwitchBack.filters.type).toBe('map');
    expect(afterSwitchBack.filters.map.locations).toEqual(['europe']);
    expect(afterSwitchBack.page).toBe(2);
  });
});

describe('defaultSearchFilters vs defaultLibraryFilters', () => {
  it('defaults to map type for search', () => {
    expect(defaultSearchFilters.type).toBe('map');
  });

  it('defaults to name:asc sort for library', () => {
    expect(defaultLibraryFilters.sort.field).toBe('name');
    expect(defaultLibraryFilters.sort.direction).toBe('asc');
  });

  it('defaults to last_updated:desc sort for search', () => {
    expect(defaultSearchFilters.sort.field).toBe('last_updated');
    expect(defaultSearchFilters.sort.direction).toBe('desc');
  });

  it('shares the same default perPage', () => {
    expect(defaultSearchFilters.perPage).toBe(defaultLibraryFilters.perPage);
  });
});
