import { beforeEach, describe, expect, it } from 'vitest';

import { useSearchStore } from '@/stores/search-store';

describe('useSearchStore per-asset-type state', () => {
  beforeEach(() => {
    useSearchStore.setState({
      filters: {
        query: '',
        type: 'map',
        sort: { field: 'last_updated', direction: 'desc' },
        randomSeed: 100,
        perPage: 12,
        mod: { tags: [] },
        map: {
          locations: [],
          sourceQuality: [],
          levelOfDetail: [],
          specialDemand: [],
        },
      },
      page: 1,
      scopedByType: {
        map: {
          sort: { field: 'last_updated', direction: 'desc' },
          randomSeed: 100,
          mod: { tags: [] },
          map: {
            locations: [],
            sourceQuality: [],
            levelOfDetail: [],
            specialDemand: [],
          },
          page: 1,
        },
        mod: {
          sort: { field: 'last_updated', direction: 'desc' },
          randomSeed: 100,
          mod: { tags: [] },
          map: {
            locations: [],
            sourceQuality: [],
            levelOfDetail: [],
            specialDemand: [],
          },
          page: 1,
        },
      },
    });
  });

  it('restores map/mod scoped sort, filters, random seed, and page when switching type', () => {
    useSearchStore.setState((state) => ({
      ...state,
      filters: {
        ...state.filters,
        sort: { field: 'population', direction: 'desc' },
        randomSeed: 11,
        map: {
          ...state.filters.map,
          locations: ['europe'],
        },
      },
      page: 3,
    }));

    useSearchStore.getState().setType('mod');

    useSearchStore.setState((state) => ({
      ...state,
      filters: {
        ...state.filters,
        sort: { field: 'author', direction: 'asc' },
        randomSeed: 22,
        mod: {
          ...state.filters.mod,
          tags: ['ui'],
        },
      },
      page: 2,
    }));

    useSearchStore.getState().setType('map');
    let state = useSearchStore.getState();
    expect(state.filters.type).toBe('map');
    expect(state.filters.sort).toEqual({
      field: 'population',
      direction: 'desc',
    });
    expect(state.filters.randomSeed).toBe(11);
    expect(state.filters.map.locations).toEqual(['europe']);
    expect(state.page).toBe(3);

    useSearchStore.getState().setType('mod');
    state = useSearchStore.getState();
    expect(state.filters.type).toBe('mod');
    expect(state.filters.sort).toEqual({ field: 'author', direction: 'asc' });
    expect(state.filters.randomSeed).toBe(22);
    expect(state.filters.mod.tags).toEqual(['ui']);
    expect(state.page).toBe(2);
  });

  it('keeps query and perPage shared across type switches', () => {
    useSearchStore.getState().setFilters((prev) => ({
      ...prev,
      query: 'metro',
      perPage: 24,
    }));

    useSearchStore.getState().setType('mod');
    let state = useSearchStore.getState();
    expect(state.filters.query).toBe('metro');
    expect(state.filters.perPage).toBe(24);

    useSearchStore.getState().setType('map');
    state = useSearchStore.getState();
    expect(state.filters.query).toBe('metro');
    expect(state.filters.perPage).toBe(24);
  });
});
