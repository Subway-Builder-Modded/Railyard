import { beforeEach, describe, expect, it } from 'vitest';

import { useBrowseStore } from '@/stores/browse-store';

describe('useBrowseStore per-asset-type state', () => {
  beforeEach(() => {
    useBrowseStore.setState({
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
      viewMode: 'full',
      viewModeInitialized: false,
    });
  });

  it('restores map/mod scoped sort, filters, random seed, and page when switching type', () => {
    useBrowseStore.setState((state) => ({
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

    useBrowseStore.getState().setType('mod');

    useBrowseStore.setState((state) => ({
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

    useBrowseStore.getState().setType('map');
    let state = useBrowseStore.getState();
    expect(state.filters.type).toBe('map');
    expect(state.filters.sort).toEqual({
      field: 'population',
      direction: 'desc',
    });
    expect(state.filters.randomSeed).toBe(11);
    expect(state.filters.map.locations).toEqual(['europe']);
    expect(state.page).toBe(3);

    useBrowseStore.getState().setType('mod');
    state = useBrowseStore.getState();
    expect(state.filters.type).toBe('mod');
    expect(state.filters.sort).toEqual({ field: 'author', direction: 'asc' });
    expect(state.filters.randomSeed).toBe(22);
    expect(state.filters.mod.tags).toEqual(['ui']);
    expect(state.page).toBe(2);
  });

  it('keeps query and perPage shared across type switches', () => {
    useBrowseStore.getState().setFilters((prev) => ({
      ...prev,
      query: 'metro',
      perPage: 24,
    }));

    useBrowseStore.getState().setType('mod');
    let state = useBrowseStore.getState();
    expect(state.filters.query).toBe('metro');
    expect(state.filters.perPage).toBe(24);

    useBrowseStore.getState().setType('map');
    state = useBrowseStore.getState();
    expect(state.filters.query).toBe('metro');
    expect(state.filters.perPage).toBe(24);
  });
});

describe('useBrowseStore view mode', () => {
  beforeEach(() => {
    useBrowseStore.setState({
      viewMode: 'full',
      viewModeInitialized: false,
    });
  });

  it('setViewMode updates viewMode and marks initialized', () => {
    useBrowseStore.getState().setViewMode('compact');
    expect(useBrowseStore.getState().viewMode).toBe('compact');
    expect(useBrowseStore.getState().viewModeInitialized).toBe(true);
  });

  it('initializeViewMode sets mode when not yet initialized', () => {
    useBrowseStore.getState().initializeViewMode('list');
    expect(useBrowseStore.getState().viewMode).toBe('list');
    expect(useBrowseStore.getState().viewModeInitialized).toBe(true);
  });

  it('initializeViewMode is a no-op after already initialized', () => {
    useBrowseStore.setState({ viewMode: 'compact', viewModeInitialized: true });
    useBrowseStore.getState().initializeViewMode('list');
    expect(useBrowseStore.getState().viewMode).toBe('compact');
  });

  it('setViewMode always overrides even when already initialized', () => {
    useBrowseStore.setState({ viewMode: 'compact', viewModeInitialized: true });
    useBrowseStore.getState().setViewMode('list');
    expect(useBrowseStore.getState().viewMode).toBe('list');
  });
});
