import { describe, expect, it } from 'vitest';

import {
  getSortOptionsForType,
  SORT_OPTIONS,
  SortKey as SortKeyHelper,
  sortKeyToState,
  type SortState,
  sortStateToOptionKey,
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
