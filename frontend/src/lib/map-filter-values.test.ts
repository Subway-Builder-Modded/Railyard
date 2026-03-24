import { describe, expect, it } from 'vitest';

import {
  buildSpecialDemandValues,
  formatSourceQuality,
  LEVEL_OF_DETAIL_VALUES,
  LOCATION_TAGS,
  SOURCE_QUALITY_VALUES,
} from './map-filter-values';

describe('LOCATION_TAGS', () => {
  it('is a non-empty list of region strings', () => {
    expect(LOCATION_TAGS.length).toBeGreaterThan(0);
    expect(LOCATION_TAGS).toContain('europe');
    expect(LOCATION_TAGS).toContain('north-america');
  });
});

describe('SOURCE_QUALITY_VALUES', () => {
  it('contains low, medium, and high quality options', () => {
    expect(SOURCE_QUALITY_VALUES).toContain('low-quality');
    expect(SOURCE_QUALITY_VALUES).toContain('medium-quality');
    expect(SOURCE_QUALITY_VALUES).toContain('high-quality');
  });
});

describe('LEVEL_OF_DETAIL_VALUES', () => {
  it('contains low, medium, and high detail options', () => {
    expect(LEVEL_OF_DETAIL_VALUES).toContain('low-detail');
    expect(LEVEL_OF_DETAIL_VALUES).toContain('medium-detail');
    expect(LEVEL_OF_DETAIL_VALUES).toContain('high-detail');
  });
});

describe('formatSourceQuality', () => {
  it('maps known values to their display labels', () => {
    expect(formatSourceQuality('low-quality')).toBe('low-data-quality');
    expect(formatSourceQuality('medium-quality')).toBe('medium-data-quality');
    expect(formatSourceQuality('high-quality')).toBe('high-data-quality');
  });

  it('returns the original value for unknown strings', () => {
    expect(formatSourceQuality('unknown')).toBe('unknown');
  });
});

describe('buildSpecialDemandValues', () => {
  it('collects and deduplicates special demand values across maps', () => {
    const values = buildSpecialDemandValues([
      { special_demand: ['tram', 'metro'] },
      { special_demand: ['tram', 'ferry'] },
    ]);
    expect(values).toEqual(['ferry', 'metro', 'tram']); // sorted
  });

  it('returns empty array when no maps have special demand', () => {
    expect(buildSpecialDemandValues([{}, { special_demand: null }])).toEqual(
      [],
    );
  });

  it('returns empty array for empty input', () => {
    expect(buildSpecialDemandValues([])).toEqual([]);
  });
});
