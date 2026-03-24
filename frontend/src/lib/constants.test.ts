import { describe, expect, it } from 'vitest';

import {
  getSortOptionsForType,
  normalizeSortStateForType,
  SORT_OPTIONS,
  SortKey as SortKeyHelper,
  sortKeyToState,
  type SortState,
  sortStateToOptionKey,
  TEXT_SORT_FIELDS,
  toggleSortField,
} from './constants';

describe('sort helpers', () => {
  it('maps sort key to structured state', () => {
    expect(sortKeyToState('downloads:desc')).toEqual({
      field: 'downloads',
      direction: 'desc',
    });
    expect(sortKeyToState('last_updated:desc')).toEqual({
      field: 'last_updated',
      direction: 'desc',
    });
    expect(sortKeyToState('random:asc')).toEqual({
      field: 'random',
      direction: 'asc',
    });
  });

  it('maps structured state to sort key', () => {
    const state: SortState = { field: 'downloads', direction: 'asc' };
    expect(sortStateToOptionKey(state, 'mod')).toBe('downloads:asc');
  });

  it('compares sort keys via helper', () => {
    expect(SortKeyHelper.equals('downloads:asc', 'downloads:asc')).toBe(true);
    expect(SortKeyHelper.equals('downloads:asc', 'downloads:desc')).toBe(false);
  });

  it('hides population options for mods only', () => {
    const modOptions = getSortOptionsForType('mod');
    const mapOptions = getSortOptionsForType('map');

    expect(modOptions).toHaveLength(9);
    expect(modOptions.map((opt) => opt.value)).not.toContain('population:asc');
    expect(modOptions.map((opt) => opt.value)).not.toContain('population:desc');
    expect(modOptions.map((opt) => opt.value)).not.toContain('city_code:asc');
    expect(modOptions.map((opt) => opt.value)).not.toContain('city_code:desc');
    expect(modOptions.map((opt) => opt.value)).toContain('last_updated:asc');
    expect(modOptions.map((opt) => opt.value)).toContain('last_updated:desc');
    expect(modOptions.map((opt) => opt.value)).toContain('random:asc');
    expect(modOptions.map((opt) => opt.value)).not.toContain('random:desc');
    expect(mapOptions).toHaveLength(15);
    expect(mapOptions).toEqual(SORT_OPTIONS);
  });

  it('orders alphabetical sort keys as asc before desc', () => {
    const mapOptions = getSortOptionsForType('map');
    const values = mapOptions.map((option) => option.value);

    expect(values.indexOf('name:asc')).toBeLessThan(
      values.indexOf('name:desc'),
    );
    expect(values.indexOf('author:asc')).toBeLessThan(
      values.indexOf('author:desc'),
    );
    expect(values.indexOf('country:asc')).toBeLessThan(
      values.indexOf('country:desc'),
    );
    expect(values.indexOf('city_code:asc')).toBeLessThan(
      values.indexOf('city_code:desc'),
    );
  });

  it('falls back to default when sort key is invalid', () => {
    expect(sortKeyToState('nope')).toEqual({
      field: 'last_updated',
      direction: 'desc',
    });
  });
});

describe('toggleSortField', () => {
  it('inverts direction when the same field is toggled', () => {
    expect(
      toggleSortField({ field: 'name', direction: 'asc' }, 'name'),
    ).toEqual({ field: 'name', direction: 'desc' });
    expect(
      toggleSortField({ field: 'downloads', direction: 'desc' }, 'downloads'),
    ).toEqual({ field: 'downloads', direction: 'asc' });
  });

  it('resets direction to asc when a different field is toggled', () => {
    expect(
      toggleSortField({ field: 'name', direction: 'desc' }, 'author'),
    ).toEqual({ field: 'author', direction: 'asc' });
    expect(
      toggleSortField({ field: 'downloads', direction: 'asc' }, 'name'),
    ).toEqual({ field: 'name', direction: 'asc' });
  });
});

describe('normalizeSortStateForType', () => {
  it('returns the same state when it is valid for the type', () => {
    const modState: SortState = { field: 'downloads', direction: 'desc' };
    expect(normalizeSortStateForType(modState, 'mod')).toEqual(modState);

    const mapState: SortState = { field: 'population', direction: 'asc' };
    expect(normalizeSortStateForType(mapState, 'map')).toEqual(mapState);
  });

  it('falls back when a map-only field is requested for mod type', () => {
    const normalized = normalizeSortStateForType(
      { field: 'population', direction: 'desc' },
      'mod',
    );
    expect(normalized.field).not.toBe('population');
    expect(normalized.field).not.toBe('city_code');
    expect(normalized.field).not.toBe('country');
  });

  it('allows map-only fields for map type', () => {
    expect(
      normalizeSortStateForType(
        { field: 'population', direction: 'asc' },
        'map',
      ),
    ).toEqual({ field: 'population', direction: 'asc' });
    expect(
      normalizeSortStateForType(
        { field: 'city_code', direction: 'desc' },
        'map',
      ),
    ).toEqual({ field: 'city_code', direction: 'desc' });
  });

  it('falls back gracefully for an entirely unknown field', () => {
    const normalized = normalizeSortStateForType(
      { field: 'nope' as never, direction: 'asc' },
      'mod',
    );
    expect(normalized).toBeDefined();
    expect(normalized.field).toBeTruthy();
  });
});

describe('TEXT_SORT_FIELDS', () => {
  it('contains only alphabetically-sorted fields', () => {
    expect(TEXT_SORT_FIELDS.has('name')).toBe(true);
    expect(TEXT_SORT_FIELDS.has('author')).toBe(true);
    expect(TEXT_SORT_FIELDS.has('city_code')).toBe(true);
    expect(TEXT_SORT_FIELDS.has('country')).toBe(true);
  });

  it('does not contain numeric or non-text fields', () => {
    expect(TEXT_SORT_FIELDS.has('downloads')).toBe(false);
    expect(TEXT_SORT_FIELDS.has('population')).toBe(false);
    expect(TEXT_SORT_FIELDS.has('random')).toBe(false);
    expect(TEXT_SORT_FIELDS.has('last_updated')).toBe(false);
  });
});
