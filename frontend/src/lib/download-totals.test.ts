import { describe, expect, it } from 'vitest';

import {
  sumVersionDownloads,
  toCumulativeDownloadTotals,
} from './download-totals';

describe('sumVersionDownloads', () => {
  it('sums all version counts for an asset', () => {
    expect(sumVersionDownloads({ '1.0.0': 12, '1.1.0': 8 })).toBe(20);
  });

  it('returns zero when counts are missing', () => {
    expect(sumVersionDownloads(undefined)).toBe(0);
  });

  it('returns zero for an empty object', () => {
    expect(sumVersionDownloads({})).toBe(0);
  });

  it('ignores non-finite values', () => {
    expect(
      sumVersionDownloads({ '1.0.0': NaN, '1.1.0': Infinity, '1.2.0': 5 }),
    ).toBe(5);
  });
});

describe('toCumulativeDownloadTotals', () => {
  it('builds cumulative totals per asset', () => {
    expect(
      toCumulativeDownloadTotals({
        map_a: { '1.0.0': 2, '1.1.0': 3 },
        map_b: { '2.0.0': 7 },
      }),
    ).toEqual({ map_a: 5, map_b: 7 });
  });

  it('returns empty object when input is undefined', () => {
    expect(toCumulativeDownloadTotals(undefined)).toEqual({});
  });

  it('returns empty object for empty input', () => {
    expect(toCumulativeDownloadTotals({})).toEqual({});
  });
});
