import { describe, expect, it } from 'vitest';

import {
  buildTaggedItems,
  compareByDirection,
  compareItems,
  getLastUpdated,
  getTotalDownloads,
  sortTaggedItemsByLastUpdated,
  type TaggedItem,
} from './tagged-items';

// Minimal fixture helpers — use plain objects since tagged-items only uses
// `import type` for the wailsjs models, so no class instantiation needed at runtime.
function makeMod(overrides: Record<string, unknown> = {}): TaggedItem {
  return {
    type: 'mod',
    item: {
      id: 'mod-1',
      name: 'Test Mod',
      author: 'Author',
      last_updated: 1000,
      ...overrides,
    } as never,
  };
}

function makeMap(overrides: Record<string, unknown> = {}): TaggedItem {
  return {
    type: 'map',
    item: {
      id: 'map-1',
      name: 'Test Map',
      author: 'Author',
      last_updated: 2000,
      city_code: 'TST',
      country: 'testland',
      population: 100_000,
      ...overrides,
    } as never,
  };
}

describe('buildTaggedItems', () => {
  it('combines mods and maps into a flat tagged list', () => {
    const items = buildTaggedItems(
      [{ id: 'mod-1' } as never],
      [{ id: 'map-1' } as never],
    );
    expect(items).toHaveLength(2);
    expect(items[0].type).toBe('mod');
    expect(items[1].type).toBe('map');
  });

  it('returns empty array when both lists are empty', () => {
    expect(buildTaggedItems([], [])).toHaveLength(0);
  });
});

describe('compareByDirection', () => {
  it('returns negative for ascending order when a < b', () => {
    expect(compareByDirection(1, 2, 'asc')).toBeLessThan(0);
  });

  it('returns positive for ascending order when a > b', () => {
    expect(compareByDirection(2, 1, 'asc')).toBeGreaterThan(0);
  });

  it('inverts for descending order', () => {
    expect(compareByDirection(1, 2, 'desc')).toBeGreaterThan(0);
    expect(compareByDirection(2, 1, 'desc')).toBeLessThan(0);
  });
});

describe('getTotalDownloads', () => {
  it('returns mod download total from modDownloadTotals', () => {
    const item = makeMod({ id: 'mod-a' });
    expect(getTotalDownloads(item, { 'mod-a': 42 }, {})).toBe(42);
  });

  it('returns map download total from mapDownloadTotals', () => {
    const item = makeMap({ id: 'map-a' });
    expect(getTotalDownloads(item, {}, { 'map-a': 99 })).toBe(99);
  });

  it('returns zero when id is not in totals', () => {
    expect(getTotalDownloads(makeMod(), {}, {})).toBe(0);
    expect(getTotalDownloads(makeMap(), {}, {})).toBe(0);
  });
});

describe('getLastUpdated', () => {
  it('returns the last_updated timestamp', () => {
    expect(getLastUpdated(makeMod({ last_updated: 9999 }))).toBe(9999);
  });

  it('returns zero for non-finite timestamps', () => {
    expect(getLastUpdated(makeMod({ last_updated: NaN }))).toBe(0);
    expect(getLastUpdated(makeMod({ last_updated: null }))).toBe(0);
    expect(getLastUpdated(makeMod({ last_updated: undefined }))).toBe(0);
  });
});

describe('compareItems', () => {
  it('sorts by name alphabetically', () => {
    const a = makeMod({ name: 'Alpha' });
    const b = makeMod({ name: 'Zebra' });
    expect(
      compareItems(a, b, { field: 'name', direction: 'asc' }, {}, {}),
    ).toBeLessThan(0);
    expect(
      compareItems(a, b, { field: 'name', direction: 'desc' }, {}, {}),
    ).toBeGreaterThan(0);
  });

  it('sorts by author alphabetically', () => {
    const a = makeMod({ author: 'Alice' });
    const b = makeMod({ author: 'Zara' });
    expect(
      compareItems(a, b, { field: 'author', direction: 'asc' }, {}, {}),
    ).toBeLessThan(0);
  });

  it('sorts by downloads using totals lookup', () => {
    const a = makeMod({ id: 'mod-a' });
    const b = makeMod({ id: 'mod-b' });
    const result = compareItems(
      a,
      b,
      { field: 'downloads', direction: 'desc' },
      { 'mod-a': 100, 'mod-b': 50 },
      {},
    );
    expect(result).toBeLessThan(0); // a has more downloads, comes first descending
  });

  it('sorts by last_updated', () => {
    const older = makeMod({ last_updated: 100 });
    const newer = makeMod({ last_updated: 999 });
    expect(
      compareItems(
        older,
        newer,
        { field: 'last_updated', direction: 'desc' },
        {},
        {},
      ),
    ).toBeGreaterThan(0); // newer should come first
  });

  it('sorts maps by population', () => {
    const small = makeMap({ population: 10_000 });
    const large = makeMap({ population: 1_000_000 });
    expect(
      compareItems(
        small,
        large,
        { field: 'population', direction: 'desc' },
        {},
        {},
      ),
    ).toBeGreaterThan(0);
  });

  it('sorts maps by city_code', () => {
    const a = makeMap({ city_code: 'AAA' });
    const b = makeMap({ city_code: 'ZZZ' });
    expect(
      compareItems(a, b, { field: 'city_code', direction: 'asc' }, {}, {}),
    ).toBeLessThan(0);
  });

  it('sorts maps by country', () => {
    const a = makeMap({ country: 'austria' });
    const b = makeMap({ country: 'zimbabwe' });
    expect(
      compareItems(a, b, { field: 'country', direction: 'asc' }, {}, {}),
    ).toBeLessThan(0);
  });
});

describe('sortTaggedItemsByLastUpdated', () => {
  it('sorts descending by default', () => {
    const items = [
      makeMod({ id: 'mod-old', last_updated: 100 }),
      makeMod({ id: 'mod-new', last_updated: 999 }),
    ];
    const sorted = sortTaggedItemsByLastUpdated(items);
    expect((sorted[0].item as { id: string }).id).toBe('mod-new');
  });

  it('can sort ascending', () => {
    const items = [
      makeMod({ id: 'mod-old', last_updated: 100 }),
      makeMod({ id: 'mod-new', last_updated: 999 }),
    ];
    const sorted = sortTaggedItemsByLastUpdated(items, 'asc');
    expect((sorted[0].item as { id: string }).id).toBe('mod-old');
  });

  it('does not mutate the original array', () => {
    const items = [
      makeMod({ id: 'mod-a', last_updated: 500 }),
      makeMod({ id: 'mod-b', last_updated: 100 }),
    ];
    const original = [...items];
    sortTaggedItemsByLastUpdated(items);
    expect(items).toEqual(original);
  });
});
