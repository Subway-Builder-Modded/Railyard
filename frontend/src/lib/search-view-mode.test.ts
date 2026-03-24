import { describe, expect, it } from 'vitest';

import {
  isSearchViewMode,
  normalizeSearchViewMode,
  SEARCH_VIEW_MODES,
} from './search-view-mode';

describe('SEARCH_VIEW_MODES', () => {
  it('contains full, compact, and list', () => {
    expect(SEARCH_VIEW_MODES).toContain('full');
    expect(SEARCH_VIEW_MODES).toContain('compact');
    expect(SEARCH_VIEW_MODES).toContain('list');
  });
});

describe('isSearchViewMode', () => {
  it('returns true for valid view mode strings', () => {
    expect(isSearchViewMode('full')).toBe(true);
    expect(isSearchViewMode('compact')).toBe(true);
    expect(isSearchViewMode('list')).toBe(true);
  });

  it('returns false for unknown strings', () => {
    expect(isSearchViewMode('grid')).toBe(false);
    expect(isSearchViewMode('')).toBe(false);
  });

  it('returns false for non-string types', () => {
    expect(isSearchViewMode(null)).toBe(false);
    expect(isSearchViewMode(undefined)).toBe(false);
    expect(isSearchViewMode(42)).toBe(false);
    expect(isSearchViewMode({})).toBe(false);
  });
});

describe('normalizeSearchViewMode', () => {
  it('returns the value as-is when it is a valid view mode', () => {
    expect(normalizeSearchViewMode('compact')).toBe('compact');
    expect(normalizeSearchViewMode('list')).toBe('list');
  });

  it('falls back to "full" for invalid values', () => {
    expect(normalizeSearchViewMode('grid')).toBe('full');
    expect(normalizeSearchViewMode(undefined)).toBe('full');
    expect(normalizeSearchViewMode(null)).toBe('full');
  });

  it('uses the provided fallback when given', () => {
    expect(normalizeSearchViewMode('nope', 'compact')).toBe('compact');
  });
});
